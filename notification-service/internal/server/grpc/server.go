package grpc

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/core"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	pb "github.com/MuhibNayem/connectify-v2/notification-service/proto/notification/v1"
	"google.golang.org/grpc"
)

type NotificationServer struct {
	pb.UnimplementedNotificationServiceServer
	orchestrator *core.Orchestrator
}

func NewNotificationServer(orchestrator *core.Orchestrator) *NotificationServer {
	return &NotificationServer{
		orchestrator: orchestrator,
	}
}

func (s *NotificationServer) Register(grpcServer *grpc.Server) {
	pb.RegisterNotificationServiceServer(grpcServer, s)
}

func (s *NotificationServer) CreateNotification(ctx context.Context, req *pb.CreateNotificationRequest) (*pb.CreateNotificationResponse, error) {
	// Convert proto request to internal model
	data := make(map[string]interface{})
	for k, v := range req.Data {
		data[k] = v
	}

	// Create request model
	reqModel := &models.CreateNotificationRequest{
		RecipientID: req.RecipientId,
		Type:        req.Type,
		Title:       req.Title,
		Body:        req.Body,
		Priority:    models.Priority(req.Priority),
		Data:        data,
	}

	result, err := s.orchestrator.CreateNotification(ctx, reqModel)
	if err != nil {
		return nil, err
	}

	return &pb.CreateNotificationResponse{
		Id:      result.ID,
		Success: true,
	}, nil
}

func (s *NotificationServer) GetNotification(ctx context.Context, req *pb.GetNotificationRequest) (*pb.GetNotificationResponse, error) {
	notif, err := s.orchestrator.GetNotification(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.GetNotificationResponse{
		Notification: s.toProto(notif),
	}, nil
}

func (s *NotificationServer) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	reqModel := &models.ListNotificationsRequest{
		RecipientID: req.RecipientId,
		Limit:       int(req.Limit),
		// Helper to convert Offset to Cursor if needed, or update core to support Offset
		// For now assuming existing core supports cursor, we might need to adjust generic ListQuery
	}

	result, err := s.orchestrator.ListNotifications(ctx, reqModel)
	if err != nil {
		return nil, err
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
