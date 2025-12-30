package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/auth"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/core"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/ratelimit"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	orchestrator  *core.Orchestrator
	router        *chi.Mux
	authenticator *auth.Authenticator
	rateLimiter   *ratelimit.Limiter
}

// ServerConfig holds all dependencies for the HTTP server
type ServerConfig struct {
	Orchestrator  *core.Orchestrator
	Authenticator *auth.Authenticator
	RateLimiter   *ratelimit.Limiter
}

func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		orchestrator:  cfg.Orchestrator,
		router:        chi.NewRouter(),
		authenticator: cfg.Authenticator,
		rateLimiter:   cfg.RateLimiter,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check (no auth required)
	s.router.Get("/health", s.healthCheck)

	// API v1
	s.router.Route("/v1", func(r chi.Router) {
		// Wire real authentication middleware
		if s.authenticator != nil {
			r.Use(s.authenticator.Middleware)
		}

		// Wire real rate limiting middleware
		if s.rateLimiter != nil {
			r.Use(s.rateLimiter.Middleware(ratelimit.IPKeyExtractor))
		}

		r.Post("/notifications", s.createNotification)
		r.Get("/notifications", s.listNotifications)
		r.Get("/notifications/{id}", s.getNotification)
		r.Patch("/notifications/{id}", s.updateNotification)
		r.Delete("/notifications/{id}", s.deleteNotification)
		r.Post("/notifications/batch/mark-read", s.batchMarkAsRead)
		r.Get("/notifications/unread-count", s.getUnreadCount)
	})
}

// validateRecipientAccess checks if the authenticated user is allowed to access the given recipient's data.
// Returns the effective recipientID to use, or an error.
func (s *Server) validateRecipientAccess(r *http.Request, requestedRecipientID string) (string, error) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		// Should be caught by middleware, but safe fallback
		return "", http.ErrNoCookie
	}

	// Service accounts (API Keys) have full access
	if claims.Role == "service" || claims.Role == "admin" {
		if requestedRecipientID == "" {
			return "", convertError("recipient_id is required for service calls")
		}
		return requestedRecipientID, nil
	}

	// End users can only access their own data
	if requestedRecipientID != "" && requestedRecipientID != claims.UserID {
		return "", convertError("forbidden: cannot access other user's data")
	}

	return claims.UserID, nil
}

func convertError(msg string) error {
	return &httpError{msg}
}

type httpError struct {
	msg string
}

func (e *httpError) Error() string { return e.msg }

func (s *Server) checkNotificationAccess(ctx context.Context, notifID string) (*models.Notification, error) {
	notif, err := s.orchestrator.GetNotification(ctx, notifID)
	if err != nil {
		return nil, err
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims == nil {
		return nil, http.ErrNoCookie
	}

	if claims.Role == "service" || claims.Role == "admin" {
		return notif, nil
	}

	if notif.RecipientID != claims.UserID {
		return nil, convertError("forbidden: notification belongs to another user")
	}

	return notif, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "notification-service",
	})
}

func (s *Server) createNotification(w http.ResponseWriter, r *http.Request) {
	var req models.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// AuthZ: Enforce RecipientID match for non-admin users
	recipientID, err := s.validateRecipientAccess(r, req.RecipientID)
	if err != nil {
		if err == http.ErrNoCookie { // Context checked in middleware
			// Pass through if valid auth logic allows weak auth, but here we strict
		} else {
			s.errorResponse(w, http.StatusForbidden, err.Error())
			return
		}
	}
	// Force correct ID for user
	req.RecipientID = recipientID
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		req.TenantID = claims.TenantID
	}

	notif, err := s.orchestrator.CreateNotification(r.Context(), &req)
	if err != nil {
		if err == core.ErrDuplicateRequest {
			s.errorResponse(w, http.StatusConflict, "duplicate request")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notif)
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	requestedRecipientID := r.URL.Query().Get("recipient_id")

	effectiveRecipientID, err := s.validateRecipientAccess(r, requestedRecipientID)
	if err != nil {
		s.errorResponse(w, http.StatusForbidden, err.Error())
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 20
	}

	req := &models.ListNotificationsRequest{
		RecipientID: effectiveRecipientID,
		Limit:       limit,
		Cursor:      r.URL.Query().Get("cursor"),
		Sort:        r.URL.Query().Get("sort"),
	}
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		req.TenantID = claims.TenantID
	}

	// Parse read filter
	if readStr := r.URL.Query().Get("read"); readStr != "" {
		read := readStr == "true"
		req.Read = &read
	}

	result, err := s.orchestrator.ListNotifications(r.Context(), req)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) getNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		s.errorResponse(w, http.StatusBadRequest, "id is required")
		return
	}

	notif, err := s.checkNotificationAccess(r.Context(), id)
	if err != nil {
		if _, ok := err.(*httpError); ok {
			s.errorResponse(w, http.StatusForbidden, err.Error())
		} else {
			s.errorResponse(w, http.StatusNotFound, "notification not found")
		}
		return
	}

	json.NewEncoder(w).Encode(notif)
}

func (s *Server) updateNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// AuthZ Check first
	_, err := s.checkNotificationAccess(r.Context(), id)
	if err != nil {
		s.errorResponse(w, http.StatusForbidden, err.Error())
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if read, ok := updates["read"].(bool); ok && read {
		if err := s.orchestrator.MarkAsRead(r.Context(), id); err != nil {
			s.errorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	notif, _ := s.orchestrator.GetNotification(r.Context(), id)
	json.NewEncoder(w).Encode(notif)
}

func (s *Server) deleteNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// AuthZ Check
	_, err := s.checkNotificationAccess(r.Context(), id)
	if err != nil {
		s.errorResponse(w, http.StatusForbidden, err.Error())
		return
	}

	if err := s.orchestrator.DeleteNotification(r.Context(), id); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) batchMarkAsRead(w http.ResponseWriter, r *http.Request) {
	var req models.BatchMarkAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	recipientID, err := s.validateRecipientAccess(r, req.RecipientID)
	if err != nil {
		s.errorResponse(w, http.StatusForbidden, err.Error())
		return
	}
	req.RecipientID = recipientID
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		req.TenantID = claims.TenantID
	}

	count, err := s.orchestrator.BatchMarkAsRead(r.Context(), &req)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	json.NewEncoder(w).Encode(map[string]int64{"updated_count": count})
}

func (s *Server) getUnreadCount(w http.ResponseWriter, r *http.Request) {
	requestedRecipientID := r.URL.Query().Get("recipient_id")

	recipientID, err := s.validateRecipientAccess(r, requestedRecipientID)
	if err != nil {
		s.errorResponse(w, http.StatusForbidden, err.Error())
		return
	}

	count, err := s.orchestrator.GetUnreadCount(r.Context(), recipientID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	json.NewEncoder(w).Encode(map[string]int64{"count": count})
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}
