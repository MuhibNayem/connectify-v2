package grpc

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	friendshippb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/friendship/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FriendshipServer struct {
	friendshippb.UnimplementedFriendshipServiceServer
	service *service.FriendshipService
}

func NewFriendshipServer(svc *service.FriendshipService) *FriendshipServer {
	return &FriendshipServer{service: svc}
}

func (s *FriendshipServer) SendRequest(ctx context.Context, req *friendshippb.SendRequestRequest) (*friendshippb.SendRequestResponse, error) {
	requesterID, err := primitive.ObjectIDFromHex(req.RequesterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid requester ID")
	}
	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid receiver ID")
	}

	friendship, err := s.service.SendRequest(ctx, requesterID, receiverID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &friendshippb.SendRequestResponse{
		Friendship: mapFriendshipToProto(friendship),
	}, nil
}

func (s *FriendshipServer) RespondToRequest(ctx context.Context, req *friendshippb.RespondToRequestRequest) (*friendshippb.RespondToRequestResponse, error) {
	friendshipID, err := primitive.ObjectIDFromHex(req.FriendshipId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid friendship ID")
	}
	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid receiver ID")
	}

	if err := s.service.RespondToRequest(ctx, friendshipID, receiverID, req.Accept); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &friendshippb.RespondToRequestResponse{Success: true}, nil
}

func (s *FriendshipServer) ListFriendships(ctx context.Context, req *friendshippb.ListFriendshipsRequest) (*friendshippb.ListFriendshipsResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	friendships, total, err := s.service.ListFriendships(ctx, userID, models.FriendshipStatus(req.Status), req.Page, req.Limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var protoFriendships []*friendshippb.PopulatedFriendship
	for _, f := range friendships {
		protoFriendships = append(protoFriendships, mapPopulatedFriendshipToProto(&f))
	}

	return &friendshippb.ListFriendshipsResponse{
		Friendships: protoFriendships,
		Total:       total,
	}, nil
}

func (s *FriendshipServer) CheckFriendship(ctx context.Context, req *friendshippb.CheckFriendshipRequest) (*friendshippb.CheckFriendshipResponse, error) {
	userID1, err := primitive.ObjectIDFromHex(req.UserId1)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID 1")
	}
	userID2, err := primitive.ObjectIDFromHex(req.UserId2)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID 2")
	}

	areFriends, err := s.service.CheckFriendship(ctx, userID1, userID2)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &friendshippb.CheckFriendshipResponse{AreFriends: areFriends}, nil
}

func (s *FriendshipServer) Unfriend(ctx context.Context, req *friendshippb.UnfriendRequest) (*friendshippb.UnfriendResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}
	friendID, err := primitive.ObjectIDFromHex(req.FriendId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid friend ID")
	}

	if err := s.service.Unfriend(ctx, userID, friendID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &friendshippb.UnfriendResponse{Success: true}, nil
}

func (s *FriendshipServer) BlockUser(ctx context.Context, req *friendshippb.BlockUserRequest) (*friendshippb.BlockUserResponse, error) {
	blockerID, err := primitive.ObjectIDFromHex(req.BlockerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid blocker ID")
	}
	blockedID, err := primitive.ObjectIDFromHex(req.BlockedId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid blocked ID")
	}

	if err := s.service.BlockUser(ctx, blockerID, blockedID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &friendshippb.BlockUserResponse{Success: true}, nil
}

func (s *FriendshipServer) UnblockUser(ctx context.Context, req *friendshippb.UnblockUserRequest) (*friendshippb.UnblockUserResponse, error) {
	blockerID, err := primitive.ObjectIDFromHex(req.BlockerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid blocker ID")
	}
	blockedID, err := primitive.ObjectIDFromHex(req.BlockedId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid blocked ID")
	}

	if err := s.service.UnblockUser(ctx, blockerID, blockedID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &friendshippb.UnblockUserResponse{Success: true}, nil
}

func (s *FriendshipServer) GetBlockedUsers(ctx context.Context, req *friendshippb.GetBlockedUsersRequest) (*friendshippb.GetBlockedUsersResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	blockedUsers, err := s.service.GetBlockedUsers(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var protoUsers []*friendshippb.FriendshipUser
	for _, u := range blockedUsers {
		protoUsers = append(protoUsers, mapUserToProto(&u))
	}

	return &friendshippb.GetBlockedUsersResponse{BlockedUsers: protoUsers}, nil
}

func (s *FriendshipServer) GetDetailedFriendshipStatus(ctx context.Context, req *friendshippb.GetDetailedFriendshipStatusRequest) (*friendshippb.GetDetailedFriendshipStatusResponse, error) {
	viewerID, err := primitive.ObjectIDFromHex(req.ViewerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid viewer ID")
	}
	otherUserID, err := primitive.ObjectIDFromHex(req.OtherUserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid other user ID")
	}

	statusResp, err := s.service.GetDetailedFriendshipStatus(ctx, viewerID, otherUserID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &friendshippb.GetDetailedFriendshipStatusResponse{
		AreFriends:        statusResp.AreFriends,
		RequestSent:       statusResp.RequestSent,
		RequestReceived:   statusResp.RequestReceived,
		IsBlockedByViewer: statusResp.IsBlockedByViewer,
		HasBlockedViewer:  statusResp.HasBlockedViewer,
	}, nil
}

func (s *FriendshipServer) SearchFriends(ctx context.Context, req *friendshippb.SearchFriendsRequest) (*friendshippb.SearchFriendsResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	results, err := s.service.SearchFriends(ctx, userID, req.Query, req.Limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var protoResults []*friendshippb.FriendshipUser
	for _, r := range results {
		// UserShortResponse map to FriendshipUser
		protoResults = append(protoResults, &friendshippb.FriendshipUser{
			Id:       r.ID.Hex(),
			Username: r.Username,
			FullName: r.FullName,
			Avatar:   r.Avatar,
		})
	}

	return &friendshippb.SearchFriendsResponse{Results: protoResults}, nil
}

// Helpers

func mapFriendshipToProto(f *models.Friendship) *friendshippb.Friendship {
	if f == nil {
		return nil
	}
	return &friendshippb.Friendship{
		Id:          f.ID.Hex(),
		RequesterId: f.RequesterID.Hex(),
		ReceiverId:  f.ReceiverID.Hex(),
		Status:      string(f.Status),
		CreatedAt:   timestamppb.New(f.CreatedAt),
		UpdatedAt:   timestamppb.New(f.UpdatedAt),
	}
}

func mapPopulatedFriendshipToProto(f *models.PopulatedFriendship) *friendshippb.PopulatedFriendship {
	if f == nil {
		return nil
	}
	return &friendshippb.PopulatedFriendship{
		Id:          f.ID.Hex(),
		RequesterId: f.RequesterID.Hex(),
		ReceiverId:  f.ReceiverID.Hex(),
		Status:      string(f.Status),
		RequesterInfo: &friendshippb.FriendshipUser{
			Id:       f.RequesterInfo.ID.Hex(),
			Username: f.RequesterInfo.Username,
			FullName: f.RequesterInfo.FullName,
			Avatar:   f.RequesterInfo.Avatar,
		},
		ReceiverInfo: &friendshippb.FriendshipUser{
			Id:       f.ReceiverInfo.ID.Hex(),
			Username: f.ReceiverInfo.Username,
			FullName: f.ReceiverInfo.FullName,
			Avatar:   f.ReceiverInfo.Avatar,
		},
		CreatedAt: timestamppb.New(f.CreatedAt),
		UpdatedAt: timestamppb.New(f.UpdatedAt),
	}
}

func mapUserToProto(u *models.User) *friendshippb.FriendshipUser {
	if u == nil {
		return nil
	}
	return &friendshippb.FriendshipUser{
		Id:       u.ID.Hex(),
		Username: u.Username,
		FullName: u.FullName,
		Avatar:   u.Avatar,
	}
}
