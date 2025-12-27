package http

import (
	"net/http"
	"user-service/config"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService AuthService
	cfg         *config.Config
}

func NewAuthHandler(authService AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.authService.Register(c.Request.Context(), &user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.setRefreshCookie(c, res.RefreshToken)
	c.JSON(http.StatusCreated, res)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.authService.Login(c.Request.Context(), creds.Email, creds.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.setRefreshCookie(c, res.RefreshToken)
	c.JSON(http.StatusOK, res)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshToken := h.cookieToken(c)
	if refreshToken == "" {
		var req models.RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
			return
		}
		refreshToken = req.RefreshToken
	}

	res, err := h.authService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		h.clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.setRefreshCookie(c, res.RefreshToken)
	c.JSON(http.StatusOK, res)
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	if h.cfg == nil || h.cfg.RefreshCookieName == "" {
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	maxAge := int(h.cfg.RefreshTokenTTL.Seconds())
	if token == "" {
		c.SetCookie(h.cfg.RefreshCookieName, "", -1, "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
		return
	}

	if maxAge <= 0 {
		maxAge = 0
	}
	c.SetCookie(h.cfg.RefreshCookieName, token, maxAge, "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	if h.cfg == nil || h.cfg.RefreshCookieName == "" {
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cfg.RefreshCookieName, "", -1, "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}

func (h *AuthHandler) cookieToken(c *gin.Context) string {
	if h.cfg == nil || h.cfg.RefreshCookieName == "" {
		return ""
	}
	value, err := c.Cookie(h.cfg.RefreshCookieName)
	if err != nil {
		return ""
	}
	return value
}
