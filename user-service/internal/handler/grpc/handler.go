package grpc

import (
	"context"
	"user-service/internal/service"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	pb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// ==================== READ OPERATIONS ====================

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
		User: mapModelToProto(user),
	}, nil
}

func (h *UserHandler) GetUsers(ctx context.Context, req *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	var oids []primitive.ObjectID
	for _, idStr := range req.UserIds {
		oid, err := primitive.ObjectIDFromHex(idStr)
		if err == nil {
			oids = append(oids, oid)
		}
	}

	users, err := h.userService.GetUsersByIDs(ctx, oids)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbUsers []*pb.User
	for _, u := range users {
		pbUsers = append(pbUsers, mapModelToProto(&u))
	}

	return &pb.GetUsersResponse{Users: pbUsers}, nil
}

func (h *UserHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	users, total, err := h.userService.ListUsers(ctx, page, limit, req.Search)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbUsers []*pb.User
	for _, u := range users {
		pbUsers = append(pbUsers, mapModelToProto(&u))
	}

	return &pb.ListUsersResponse{
		Users: pbUsers,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (h *UserHandler) GetUserStatus(ctx context.Context, req *pb.GetUserStatusRequest) (*pb.GetUserStatusResponse, error) {
	statusStr, lastSeen, err := h.userService.GetUserStatus(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetUserStatusResponse{
		Status:   statusStr,
		LastSeen: lastSeen,
	}, nil
}

func (h *UserHandler) GetUsersPresence(ctx context.Context, req *pb.GetUsersPresenceRequest) (*pb.GetUsersPresenceResponse, error) {
	presenceMap, err := h.userService.GetUsersPresence(ctx, req.UserIds)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make(map[string]*pb.UserPresence)
	for userID, data := range presenceMap {
		statusStr, _ := data["status"].(string)
		lastSeen, _ := data["last_seen"].(int64)
		result[userID] = &pb.UserPresence{
			Status:   statusStr,
			LastSeen: lastSeen,
		}
	}

	return &pb.GetUsersPresenceResponse{Presence: result}, nil
}

// ==================== WRITE OPERATIONS ====================

func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	update := bson.M{}
	if req.Username != "" {
		update["username"] = req.Username
	}
	if req.Email != "" {
		update["email"] = req.Email
	}
	if req.FullName != "" {
		update["full_name"] = req.FullName
	}
	if req.Bio != "" {
		update["bio"] = req.Bio
	}
	if req.Avatar != "" {
		update["avatar"] = req.Avatar
	}
	if req.CoverPicture != "" {
		update["cover_picture"] = req.CoverPicture
	}
	if req.Gender != "" {
		update["gender"] = req.Gender
	}
	if req.Location != "" {
		update["location"] = req.Location
	}
	if req.PhoneNumber != "" {
		update["phone_number"] = req.PhoneNumber
	}
	if req.DateOfBirth != nil {
		update["date_of_birth"] = req.DateOfBirth.AsTime()
	}
	if req.IsEncryptionEnabled != nil {
		update["is_encryption_enabled"] = *req.IsEncryptionEnabled
	}

	if len(update) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	user, err := h.userService.UpdateUser(ctx, oid, update)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateUserResponse{User: mapModelToProto(user)}, nil
}

func (h *UserHandler) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.UpdateEmailResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := h.userService.UpdateEmail(ctx, oid, req.NewEmail); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateEmailResponse{Success: true}, nil
}

func (h *UserHandler) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := h.userService.UpdatePassword(ctx, oid, req.CurrentPassword, req.NewPassword); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdatePasswordResponse{Success: true}, nil
}

func (h *UserHandler) ToggleTwoFactor(ctx context.Context, req *pb.ToggleTwoFactorRequest) (*pb.ToggleTwoFactorResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := h.userService.ToggleTwoFactor(ctx, oid, req.Enable); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ToggleTwoFactorResponse{Success: true}, nil
}

func (h *UserHandler) DeactivateAccount(ctx context.Context, req *pb.DeactivateAccountRequest) (*pb.DeactivateAccountResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := h.userService.DeactivateAccount(ctx, oid); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.DeactivateAccountResponse{Success: true}, nil
}

func (h *UserHandler) UpdatePublicKey(ctx context.Context, req *pb.UpdatePublicKeyRequest) (*pb.UpdatePublicKeyResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := h.userService.UpdatePublicKey(ctx, oid, req.PublicKey, req.EncryptedPrivateKey, req.KeyBackupIv, req.KeyBackupSalt); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdatePublicKeyResponse{Success: true}, nil
}

func (h *UserHandler) UpdatePrivacySettings(ctx context.Context, req *pb.UpdatePrivacySettingsRequest) (*pb.UpdatePrivacySettingsResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	settings := &models.UpdatePrivacySettingsRequest{
		DefaultPostPrivacy:      models.PrivacySettingType(req.DefaultPostPrivacy),
		CanSeeMyFriendsList:     models.PrivacySettingType(req.CanSeeMyFriendsList),
		CanSendMeFriendRequests: models.PrivacySettingType(req.CanSendMeFriendRequests),
		CanTagMeInPosts:         models.PrivacySettingType(req.CanTagMeInPosts),
	}

	if err := h.userService.UpdatePrivacySettings(ctx, oid, settings); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdatePrivacySettingsResponse{Success: true}, nil
}

func (h *UserHandler) UpdateNotificationSettings(ctx context.Context, req *pb.UpdateNotificationSettingsRequest) (*pb.UpdateNotificationSettingsResponse, error) {
	oid, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	settings := &models.UpdateNotificationSettingsRequest{
		EmailNotifications:    req.EmailNotifications,
		PushNotifications:     req.PushNotifications,
		NotifyOnFriendRequest: req.NotifyOnFriendRequest,
		NotifyOnComment:       req.NotifyOnComment,
		NotifyOnLike:          req.NotifyOnLike,
		NotifyOnTag:           req.NotifyOnTag,
		NotifyOnMessage:       req.NotifyOnMessage,
	}

	if err := h.userService.UpdateNotificationSettings(ctx, oid, settings); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateNotificationSettingsResponse{Success: true}, nil
}

// ==================== HELPERS ====================

func mapModelToProto(user *models.User) *pb.User {
	if user == nil {
		return nil
	}

	pbUser := &pb.User{
		Id:                  user.ID.Hex(),
		Username:            user.Username,
		Email:               user.Email,
		FullName:            user.FullName,
		Avatar:              user.Avatar,
		CoverPicture:        user.CoverPicture,
		Bio:                 user.Bio,
		Role:                "",
		IsVerified:          user.EmailVerified,
		Gender:              user.Gender,
		Location:            user.Location,
		PhoneNumber:         user.PhoneNumber,
		TwoFactorEnabled:    user.TwoFactorEnabled,
		EmailVerified:       user.EmailVerified,
		IsActive:            user.IsActive,
		IsEncryptionEnabled: user.IsEncryptionEnabled,
		CreatedAt:           timestamppb.New(user.CreatedAt),
		UpdatedAt:           timestamppb.New(user.UpdatedAt),
		PrivacySettings: &pb.PrivacySettings{
			DefaultPostPrivacy:      string(user.PrivacySettings.DefaultPostPrivacy),
			CanSeeMyFriendsList:     string(user.PrivacySettings.CanSeeMyFriendsList),
			CanSendMeFriendRequests: string(user.PrivacySettings.CanSendMeFriendRequests),
			CanTagMeInPosts:         string(user.PrivacySettings.CanTagMeInPosts),
			LastUpdated:             timestamppb.New(user.PrivacySettings.LastUpdated),
		},
		NotificationSettings: &pb.NotificationSettings{
			EmailNotifications:    user.NotificationSettings.EmailNotifications,
			PushNotifications:     user.NotificationSettings.PushNotifications,
			NotifyOnFriendRequest: user.NotificationSettings.NotifyOnFriendRequest,
			NotifyOnComment:       user.NotificationSettings.NotifyOnComment,
			NotifyOnLike:          user.NotificationSettings.NotifyOnLike,
			NotifyOnTag:           user.NotificationSettings.NotifyOnTag,
			NotifyOnMessage:       user.NotificationSettings.NotifyOnMessage,
			NotifyOnBirthday:      user.NotificationSettings.NotifyOnBirthday,
			NotifyOnEventInvite:   user.NotificationSettings.NotifyOnEventInvite,
		},
	}

	if user.DateOfBirth != nil {
		pbUser.DateOfBirth = timestamppb.New(*user.DateOfBirth)
	}

	return pbUser
}
