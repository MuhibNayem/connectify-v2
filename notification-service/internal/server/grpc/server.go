package grpc

import (
	"context"
	"io"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/auth"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/core"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	pb "github.com/MuhibNayem/connectify-v2/notification-service/proto/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NotificationServer struct {
	pb.UnimplementedNotificationServiceServer
	orchestrator *core.Orchestrator
	authEnabled  bool
}

func NewNotificationServer(orchestrator *core.Orchestrator, authEnabled bool) *NotificationServer {
	return &NotificationServer{
		orchestrator: orchestrator,
		authEnabled:  authEnabled,
	}
}

func (s *NotificationServer) Register(grpcServer *grpc.Server) {
	pb.RegisterNotificationServiceServer(grpcServer, s)
}

func (s *NotificationServer) CreateNotification(ctx context.Context, req *pb.CreateNotificationRequest) (*pb.CreateNotificationResponse, error) {
	reqModel, err := s.buildCreateRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	result, err := s.orchestrator.CreateNotification(ctx, reqModel)
	if err != nil {
		if err == core.ErrDuplicateRequest {
			return nil, status.Error(codes.AlreadyExists, "duplicate request")
		}
		return nil, status.Errorf(codes.Internal, "failed to create notification: %v", err)
	}

	return &pb.CreateNotificationResponse{
		Id:      result.ID,
		Success: true,
	}, nil
}

const (
	batchSize          = 100
	batchFlushInterval = 50 * time.Millisecond
)

func (s *NotificationServer) StreamNotifications(stream pb.NotificationService_StreamNotificationsServer) error {
	// Delegate to processStream for batch processing with fail-safe guarantees.
	return s.processStream(stream, batchSize, batchFlushInterval)
}

func (s *NotificationServer) processStream(stream pb.NotificationService_StreamNotificationsServer, batchSize int, interval time.Duration) error {
	ctx := stream.Context()
	buffer := make([]*models.CreateNotificationRequest, 0, batchSize)

	reqChan := make(chan *pb.CreateNotificationRequest)
	errChan := make(chan error)

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}
			reqChan <- req
		}
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Flush Helper
	flush := func() error {
		if len(buffer) == 0 {
			return nil
		}

		results, err := s.orchestrator.CreateNotificationBatch(ctx, buffer)
		if err != nil {
			// Fail Batch: Atomic failure.
			// Since CreateBatchWithOutbox uses a transaction, failure means NO data was inserted.
			// We correctly return Success: false for all items in this batch.
			// This preserves the "Fail-Safe" rule: No partial data, no ambiguity.
			for range buffer {
				if sendErr := stream.Send(&pb.CreateNotificationResponse{Success: false}); sendErr != nil {
					return status.Errorf(codes.Unknown, "stream send error: %v", sendErr)
				}
			}
		} else {
			// Success Batch: Atomic success.
			// Orchestrator guarantees 'results[i]' corresponds to 'buffer[i]'.
			// This is FAIL-SAFE because IDs are generated in memory and assigned to indices
			// BEFORE persistence. We rely on Go slice index correspondence, not DB return order.
			for _, res := range results {
				if sendErr := stream.Send(&pb.CreateNotificationResponse{Id: res.ID, Success: true}); sendErr != nil {
					return status.Errorf(codes.Unknown, "stream send error: %v", sendErr)
				}
			}
		}
		buffer = buffer[:0]
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req := <-reqChan:
			reqModel, err := s.buildCreateRequest(ctx, req)
			if err != nil {
				return err
			}
			buffer = append(buffer, reqModel)
			if len(buffer) >= batchSize {
				if err := flush(); err != nil {
					return err
				}
			}
		case err := <-errChan:
			if err == io.EOF {
				// Flush remaining
				return flush()
			}
			return err
		case <-ticker.C:
			if err := flush(); err != nil {
				return err
			}
		}
	}
}

func (s *NotificationServer) toCreateRequestModel(req *pb.CreateNotificationRequest) *models.CreateNotificationRequest {
	data := make(map[string]interface{})
	for k, v := range req.Data {
		data[k] = v
	}

	return &models.CreateNotificationRequest{
		Type:     req.Type,
		Title:    req.Title,
		Body:     req.Body,
		Priority: models.Priority(req.Priority),
		Data:     data,
	}
}

func (s *NotificationServer) buildCreateRequest(ctx context.Context, req *pb.CreateNotificationRequest) (*models.CreateNotificationRequest, error) {
	model := s.toCreateRequestModel(req)

	recipientID := req.RecipientId
	if s.authEnabled {
		var err error
		recipientID, err = s.enforceRecipient(ctx, req.RecipientId)
		if err != nil {
			return nil, err
		}
	} else if recipientID == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_id is required")
	}

	model.RecipientID = recipientID
	model.TenantID = s.tenantIDFromClaims(ctx)
	return model, nil
}

func (s *NotificationServer) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.GetNotificationResponse, error) {
	notif, err := s.orchestrator.GetNotification(ctx, req.Id)
	if err != nil {
		if err == adapters.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "notification not found: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to get notification: %v", err)
	}
	if err := s.checkNotificationAccess(ctx, notif); err != nil {
		return nil, err
	}

	return &pb.GetNotificationResponse{
		Notification: s.toProto(notif),
	}, nil
}

func (s *NotificationServer) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	recipientID := req.RecipientId
	if s.authEnabled {
		var err error
		recipientID, err = s.enforceRecipient(ctx, req.RecipientId)
		if err != nil {
			return nil, err
		}
	} else if recipientID == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_id is required")
	}

	reqModel := &models.ListNotificationsRequest{
		RecipientID: recipientID,
		Limit:       int(req.Limit),
		TenantID:    s.tenantIDFromClaims(ctx),
	}

	result, err := s.orchestrator.ListNotifications(ctx, reqModel)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list notifications: %v", err)
	}

	protoNotifs := make([]*pb.Notification, len(result.Notifications))
	for i, n := range result.Notifications {
		protoNotifs[i] = s.toProto(&n)
	}

	return &pb.ListNotificationsResponse{
		Notifications: protoNotifs,
		Total:         result.Total,
	}, nil
}

func (s *NotificationServer) UpdateNotification(ctx context.Context, req *pb.UpdateNotificationRequest) (*pb.UpdateNotificationResponse, error) {
	if req.MarkAsRead {
		if err := s.orchestrator.MarkAsRead(ctx, req.Id); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mark as read: %v", err)
		}
	}

	notif, err := s.orchestrator.GetNotification(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "notification not found after update: %v", err)
	}
	if err := s.checkNotificationAccess(ctx, notif); err != nil {
		return nil, err
	}

	return &pb.UpdateNotificationResponse{
		Notification: s.toProto(notif),
	}, nil
}

func (s *NotificationServer) DeleteNotification(ctx context.Context, req *pb.DeleteNotificationRequest) (*pb.DeleteNotificationResponse, error) {
	notif, err := s.orchestrator.GetNotification(ctx, req.Id)
	if err != nil {
		if err == adapters.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "notification not found: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to fetch notification: %v", err)
	}

	if err := s.checkNotificationAccess(ctx, notif); err != nil {
		return nil, err
	}

	if err := s.orchestrator.DeleteNotification(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete notification: %v", err)
	}
	return &pb.DeleteNotificationResponse{Success: true}, nil
}

func (s *NotificationServer) BatchMarkAsRead(ctx context.Context, req *pb.BatchMarkAsReadRequest) (*pb.BatchMarkAsReadResponse, error) {
	recipientID := req.RecipientId
	if s.authEnabled {
		var err error
		recipientID, err = s.enforceRecipient(ctx, req.RecipientId)
		if err != nil {
			return nil, err
		}
	} else if recipientID == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_id is required")
	}

	reqModel := &models.BatchMarkAsReadRequest{
		RecipientID:     recipientID,
		NotificationIDs: req.NotificationIds,
		MarkAllAsRead:   req.MarkAll,
		TenantID:        s.tenantIDFromClaims(ctx),
	}

	count, err := s.orchestrator.BatchMarkAsRead(ctx, reqModel)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to batch mark read: %v", err)
	}

	return &pb.BatchMarkAsReadResponse{UpdatedCount: count}, nil
}

func (s *NotificationServer) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	recipientID := req.RecipientId
	if s.authEnabled {
		var err error
		recipientID, err = s.enforceRecipient(ctx, req.RecipientId)
		if err != nil {
			return nil, err
		}
	} else if recipientID == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_id is required")
	}

	count, err := s.orchestrator.GetUnreadCount(ctx, recipientID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get unread count: %v", err)
	}
	return &pb.GetUnreadCountResponse{Count: count}, nil
}

func (s *NotificationServer) toProto(n *models.Notification) *pb.Notification {
	return &pb.Notification{
		Id:          n.ID,
		RecipientId: n.RecipientID,
		Type:        n.Type,
		Title:       n.Title,
		Body:        n.Body,
		CreatedAt:   n.CreatedAt.String(),
		Read:        n.Read,
	}
}

func (s *NotificationServer) enforceRecipient(ctx context.Context, requested string) (string, error) {
	if !s.authEnabled {
		if requested == "" {
			return "", status.Error(codes.InvalidArgument, "recipient_id is required")
		}
		return requested, nil
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims == nil {
		return "", status.Error(codes.Unauthenticated, "missing authentication context")
	}

	if claims.Role == "service" || claims.Role == "admin" {
		if requested == "" {
			return "", status.Error(codes.InvalidArgument, "recipient_id is required")
		}
		return requested, nil
	}

	if requested != "" && requested != claims.UserID {
		return "", status.Error(codes.PermissionDenied, "forbidden: cannot access other user's data")
	}

	return claims.UserID, nil
}

func (s *NotificationServer) tenantIDFromClaims(ctx context.Context) string {
	if !s.authEnabled {
		return ""
	}
	if claims := auth.ClaimsFromContext(ctx); claims != nil {
		return claims.TenantID
	}
	return ""
}

func (s *NotificationServer) checkNotificationAccess(ctx context.Context, notif *models.Notification) error {
	if !s.authEnabled {
		return nil
	}

	claims := auth.ClaimsFromContext(ctx)
	if claims == nil {
		return status.Error(codes.Unauthenticated, "missing authentication context")
	}

	if claims.Role == "service" || claims.Role == "admin" {
		return nil
	}

	if notif.RecipientID != claims.UserID {
		return status.Error(codes.PermissionDenied, "forbidden: notification belongs to another user")
	}
	if notif.TenantID != "" && claims.TenantID != "" && notif.TenantID != claims.TenantID {
		return status.Error(codes.PermissionDenied, "forbidden: tenant mismatch")
	}
	return nil
}
