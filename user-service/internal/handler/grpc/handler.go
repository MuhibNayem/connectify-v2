package grpc

import (
	"context"
	"user-service/internal/service"

	pb "gitlab.com/spydotech-group/shared-entity/proto/user/v1"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	user, err := h.userService.GetUserByID(ctx, oid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &pb.GetUserResponse{
		User: &pb.User{
			Id:       user.ID.Hex(),
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
			Avatar:   user.Avatar,
			Bio:      user.Bio,
			// Add more fields as needed
		},
	}, nil
}

func (h *UserHandler) GetUsers(ctx context.Context, req *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	// Implement Batch Fetch Logic here (Optimization)
	// For now, naive loop is better than N+1 network calls from client
	// But ideally we add GetUsersByIDs to UserService

	var users []*pb.User
	for _, idStr := range req.UserIds {
		oid, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			continue
		}

		user, err := h.userService.GetUserByID(ctx, oid)
		if err == nil {
			users = append(users, &pb.User{
				Id:       user.ID.Hex(),
				Username: user.Username,
				Email:    user.Email,
				FullName: user.FullName,
				Avatar:   user.Avatar,
			})
		}
	}

	return &pb.GetUsersResponse{Users: users}, nil
}
