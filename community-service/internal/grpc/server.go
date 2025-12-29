package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/MuhibNayem/connectify-v2/community-service/internal/service"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	communitypb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/community/v1"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	communitypb.UnimplementedCommunityServiceServer
	service *service.CommunityService
}

func NewServer(service *service.CommunityService) *Server {
	return &Server{
		service: service,
	}
}

func (s *Server) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	communitypb.RegisterCommunityServiceServer(grpcServer, s)

	fmt.Printf("Community GRPC Server listening on port %s\n", port)
	return grpcServer.Serve(lis)
}

// Mappers

func mapCommunityToProto(c *models.Community) *communitypb.Community {
	if c == nil {
		return nil
	}
	members := make([]string, len(c.Members))
	for i, id := range c.Members {
		members[i] = id.Hex()
	}
	admins := make([]string, len(c.Admins))
	for i, id := range c.Admins {
		admins[i] = id.Hex()
	}
	pending := make([]string, len(c.PendingMembers))
	for i, id := range c.PendingMembers {
		pending[i] = id.Hex()
	}
	banned := make([]string, len(c.BannedUsers))
	for i, id := range c.BannedUsers {
		banned[i] = id.Hex()
	}

	rules := make([]*communitypb.CommunityRule, len(c.Rules))
	for i, r := range c.Rules {
		rules[i] = &communitypb.CommunityRule{
			Title:       r.Title,
			Description: r.Description,
		}
	}

	return &communitypb.Community{
		Id:             c.ID.Hex(),
		Name:           c.Name,
		Description:    c.Description,
		Slug:           c.Slug,
		Category:       c.Category,
		Avatar:         c.Avatar,
		CoverImage:     c.CoverImage,
		Privacy:        string(c.Privacy),
		Visibility:     string(c.Visibility),
		CreatorId:      c.CreatorID.Hex(),
		Members:        members,
		Admins:         admins,
		PendingMembers: pending,
		BannedUsers:    banned,
		Settings: &communitypb.CommunitySettings{
			RequirePostApproval:  c.Settings.RequirePostApproval,
			RequireJoinApproval:  c.Settings.RequireJoinApproval,
			AllowMemberPosts:     c.Settings.AllowMemberPosts,
			ShowGroupAffiliation: c.Settings.ShowGroupAffiliation,
		},
		Rules:               rules,
		MembershipQuestions: c.MembershipQuestions,
		Stats: &communitypb.CommunityStats{
			MemberCount: c.Stats.MemberCount,
			PostCount:   c.Stats.PostCount,
		},
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func mapCommunityResponseToProto(c *models.CommunityResponse) *communitypb.Community {
	// Re-uses Community message, but populates computed fields
	// Note: models.CommunityResponse fields are flattened in Proto Community message?
	// I added is_member, is_admin etc to Proto Community message.

	// Base mapping
	// We don't have the original 'models.Community' here easily unless we reconstruct.
	// Actually mapCommunityResponseToProto is easier if we map field by field.

	rules := make([]*communitypb.CommunityRule, len(c.Rules))
	for i, r := range c.Rules {
		rules[i] = &communitypb.CommunityRule{
			Title:       r.Title,
			Description: r.Description,
		}
	}

	return &communitypb.Community{
		Id:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Slug:        c.Slug,
		Category:    c.Category,
		Avatar:      c.Avatar,
		CoverImage:  c.CoverImage,
		Privacy:     string(c.Privacy),
		Visibility:  string(c.Visibility),
		// CreatorID not in Response? Usually it is. Assume Response has it or we missed it.
		// models.CommunityResponse struct doesn't have CreatorID explicitly? (Line 91 in community.go).
		// Line 91: field CreatorID is MISSING in CommunityResponse struct I viewed earlier!
		// That's a potential bug or it's just not returned to frontend. But Proto has it.
		// I will leave it empty if missing.

		IsMember:  c.IsMember,
		IsAdmin:   c.IsAdmin,
		IsPending: c.IsPending,

		Settings: &communitypb.CommunitySettings{
			RequirePostApproval:  c.Settings.RequirePostApproval,
			RequireJoinApproval:  c.Settings.RequireJoinApproval,
			AllowMemberPosts:     c.Settings.AllowMemberPosts,
			ShowGroupAffiliation: c.Settings.ShowGroupAffiliation,
		},
		Rules:               rules,
		MembershipQuestions: c.MembershipQuestions,
		Stats: &communitypb.CommunityStats{
			MemberCount: c.Stats.MemberCount,
			PostCount:   c.Stats.PostCount,
		},
		CreatedAt: timestamppb.New(c.CreatedAt),
	}
}

func mapUserToShort(u models.User) *communitypb.UserShort {
	return &communitypb.UserShort{
		Id:       u.ID.Hex(),
		Username: u.Username,
		FullName: u.FullName,
		Avatar:   u.Avatar,
	}
}

// RPC Implementations

func (s *Server) CreateCommunity(ctx context.Context, req *communitypb.CreateCommunityRequest) (*communitypb.CreateCommunityResponse, error) {
	creatorID, err := primitive.ObjectIDFromHex(req.CreatorId)
	if err != nil {
		return nil, err
	}

	// Map proto req to models.Req
	modelReq := models.CreateCommunityRequest{
		Name:                req.Name,
		Description:         req.Description,
		Category:            req.Category,
		Avatar:              req.Avatar,
		CoverImage:          req.CoverImage,
		Privacy:             models.CommunityPrivacy(req.Privacy),
		Visibility:          models.CommunityVisibility(req.Visibility),
		RequirePostApproval: req.RequirePostApproval,
		RequireJoinApproval: req.RequireJoinApproval,
	}

	community, err := s.service.CreateCommunity(ctx, creatorID, modelReq)
	if err != nil {
		return nil, err
	}

	return &communitypb.CreateCommunityResponse{
		Community: mapCommunityToProto(community),
	}, nil
}

func (s *Server) GetCommunity(ctx context.Context, req *communitypb.GetCommunityRequest) (*communitypb.GetCommunityResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}

	var viewerID primitive.ObjectID
	if req.ViewerId != "" {
		viewerID, _ = primitive.ObjectIDFromHex(req.ViewerId)
	}

	// Use GetDetailedCommunityResponse because proto GetCommunity expects details like IsMember
	response, err := s.service.GetDetailedCommunityResponse(ctx, communityID, viewerID)
	if err != nil {
		return nil, err
	}

	return &communitypb.GetCommunityResponse{
		Community: mapCommunityResponseToProto(response),
	}, nil
}

func (s *Server) UpdateCommunity(ctx context.Context, req *communitypb.UpdateCommunityRequest) (*communitypb.UpdateCommunityResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	modelReq := models.UpdateCommunityRequest{}
	// Manual mapping of potential nil/optional fields
	if req.Name != nil {
		modelReq.Name = *req.Name
	}
	if req.Description != nil {
		modelReq.Description = *req.Description
	}
	if req.Category != nil {
		modelReq.Category = *req.Category
	}
	if req.Avatar != nil {
		modelReq.Avatar = *req.Avatar
	}
	if req.CoverImage != nil {
		modelReq.CoverImage = *req.CoverImage
	}
	if req.Privacy != nil {
		modelReq.Privacy = models.CommunityPrivacy(*req.Privacy)
	}
	if req.Visibility != nil {
		modelReq.Visibility = models.CommunityVisibility(*req.Visibility)
	}
	if req.RequirePostApproval != nil {
		modelReq.RequirePostApproval = req.RequirePostApproval
	}
	if req.RequireJoinApproval != nil {
		modelReq.RequireJoinApproval = req.RequireJoinApproval
	}
	if req.AllowMemberPosts != nil {
		modelReq.AllowMemberPosts = req.AllowMemberPosts
	}
	if req.ShowGroupAffiliation != nil {
		modelReq.ShowGroupAffiliation = req.ShowGroupAffiliation
	}

	if len(req.Rules) > 0 {
		rules := make([]models.CommunityRule, len(req.Rules))
		for i, r := range req.Rules {
			rules[i] = models.CommunityRule{Title: r.Title, Description: r.Description}
		}
		modelReq.Rules = rules
	}
	if len(req.MembershipQuestions) > 0 {
		modelReq.MembershipQuestions = req.MembershipQuestions
	}

	err = s.service.UpdateSettings(ctx, communityID, userID, modelReq)
	if err != nil {
		return nil, err
	}

	// Fetch updated
	updated, err := s.service.GetCommunity(ctx, communityID)
	if err != nil {
		return nil, err
	}

	return &communitypb.UpdateCommunityResponse{
		Community: mapCommunityToProto(updated),
	}, nil
}

func (s *Server) ListCommunities(ctx context.Context, req *communitypb.ListCommunitiesRequest) (*communitypb.ListCommunitiesResponse, error) {
	var viewerID primitive.ObjectID
	if req.ViewerId != "" {
		viewerID, _ = primitive.ObjectIDFromHex(req.ViewerId)
	}

	communities, total, err := s.service.ListCommunities(ctx, viewerID, req.Limit, req.Page, req.Query)
	if err != nil {
		return nil, err
	}

	protos := make([]*communitypb.Community, len(communities))
	for i := range communities {
		protos[i] = mapCommunityResponseToProto(&communities[i])
	}

	return &communitypb.ListCommunitiesResponse{
		Communities: protos,
		Total:       total,
	}, nil
}

func (s *Server) GetUserCommunities(ctx context.Context, req *communitypb.GetUserCommunitiesRequest) (*communitypb.GetUserCommunitiesResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	communities, err := s.service.GetUserCommunities(ctx, userID)
	if err != nil {
		return nil, err
	}

	protos := make([]*communitypb.Community, len(communities))
	for i := range communities {
		protos[i] = mapCommunityToProto(&communities[i])
	}

	return &communitypb.GetUserCommunitiesResponse{
		Communities: protos,
	}, nil
}

func (s *Server) JoinCommunity(ctx context.Context, req *communitypb.JoinCommunityRequest) (*communitypb.JoinCommunityResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	err = s.service.JoinCommunity(ctx, communityID, userID)
	if err != nil {
		return nil, err
	}

	return &communitypb.JoinCommunityResponse{Success: true}, nil
}

func (s *Server) LeaveCommunity(ctx context.Context, req *communitypb.LeaveCommunityRequest) (*communitypb.LeaveCommunityResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	err = s.service.LeaveCommunity(ctx, communityID, userID)
	if err != nil {
		return nil, err
	}

	return &communitypb.LeaveCommunityResponse{Success: true}, nil
}

func (s *Server) ApproveMember(ctx context.Context, req *communitypb.ApproveMemberRequest) (*communitypb.ApproveMemberResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}
	actorID, err := primitive.ObjectIDFromHex(req.ActorId)
	if err != nil {
		return nil, err
	}
	targetID, err := primitive.ObjectIDFromHex(req.TargetId)
	if err != nil {
		return nil, err
	}

	err = s.service.ApproveMember(ctx, communityID, actorID, targetID)
	if err != nil {
		return nil, err
	}

	return &communitypb.ApproveMemberResponse{Success: true}, nil
}

func (s *Server) RejectMember(ctx context.Context, req *communitypb.RejectMemberRequest) (*communitypb.RejectMemberResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}
	actorID, err := primitive.ObjectIDFromHex(req.ActorId)
	if err != nil {
		return nil, err
	}
	targetID, err := primitive.ObjectIDFromHex(req.TargetId)
	if err != nil {
		return nil, err
	}

	err = s.service.RejectMember(ctx, communityID, actorID, targetID)
	if err != nil {
		return nil, err
	}

	return &communitypb.RejectMemberResponse{Success: true}, nil
}

func (s *Server) GetMembers(ctx context.Context, req *communitypb.GetMembersRequest) (*communitypb.GetMembersResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}
	var viewerID primitive.ObjectID
	if req.ViewerId != "" {
		viewerID, _ = primitive.ObjectIDFromHex(req.ViewerId)
	}

	users, total, err := s.service.GetMembers(ctx, communityID, viewerID, req.Limit, req.Page)
	if err != nil {
		return nil, err
	}

	shorts := make([]*communitypb.UserShort, len(users))
	for i, u := range users {
		shorts[i] = mapUserToShort(u)
	}

	return &communitypb.GetMembersResponse{
		Members: shorts,
		Total:   total,
	}, nil
}

func (s *Server) GetAdmins(ctx context.Context, req *communitypb.GetAdminsRequest) (*communitypb.GetAdminsResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}

	users, err := s.service.GetAdmins(ctx, communityID)
	if err != nil {
		return nil, err
	}

	shorts := make([]*communitypb.UserShort, len(users))
	for i, u := range users {
		shorts[i] = mapUserToShort(u)
	}

	return &communitypb.GetAdminsResponse{
		Admins: shorts,
	}, nil
}

func (s *Server) GetPendingMembers(ctx context.Context, req *communitypb.GetPendingMembersRequest) (*communitypb.GetPendingMembersResponse, error) {
	communityID, err := primitive.ObjectIDFromHex(req.CommunityId)
	if err != nil {
		return nil, err
	}
	actorID, err := primitive.ObjectIDFromHex(req.ActorId)
	if err != nil {
		return nil, err
	}

	users, total, err := s.service.GetPendingMembers(ctx, communityID, actorID, req.Limit, req.Page)
	if err != nil {
		return nil, err
	}

	shorts := make([]*communitypb.UserShort, len(users))
	for i, u := range users {
		shorts[i] = mapUserToShort(u)
	}

	return &communitypb.GetPendingMembersResponse{
		Members: shorts,
		Total:   total,
	}, nil
}
