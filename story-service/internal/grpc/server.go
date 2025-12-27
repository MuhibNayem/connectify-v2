package grpc

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	storypb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/story/v1"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	storypb.UnimplementedStoryServiceServer
	storyService *service.StoryService
}

func NewServer(storyService *service.StoryService) *Server {
	return &Server{storyService: storyService}
}

func (s *Server) Register(grpcServer *grpc.Server) {
	storypb.RegisterStoryServiceServer(grpcServer, s)
}

func (s *Server) CreateStory(ctx context.Context, req *storypb.CreateStoryRequest) (*storypb.StoryResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	// Author info should be fetched from user-service in production
	// For now, we'll create a basic author from userID
	author := models.PostAuthor{
		ID: req.UserId,
	}

	serviceReq := service.CreateStoryRequest{
		MediaURL:       req.MediaUrl,
		MediaType:      req.MediaType,
		Privacy:        models.PrivacySettingType(req.Privacy),
		AllowedViewers: req.AllowedViewers,
		BlockedViewers: req.BlockedViewers,
	}

	story, err := s.storyService.CreateStory(ctx, userID, author, serviceReq)
	if err != nil {
		return nil, err
	}

	return &storypb.StoryResponse{Story: toProtoStory(story)}, nil
}

func (s *Server) GetStory(ctx context.Context, req *storypb.GetStoryRequest) (*storypb.StoryResponse, error) {
	storyID, err := primitive.ObjectIDFromHex(req.StoryId)
	if err != nil {
		return nil, err
	}

	viewerID, err := primitive.ObjectIDFromHex(req.ViewerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid viewer id")
	}

	story, err := s.storyService.GetStory(ctx, storyID, viewerID)
	if err != nil {
		return nil, err
	}

	return &storypb.StoryResponse{Story: toProtoStory(story)}, nil
}

func (s *Server) DeleteStory(ctx context.Context, req *storypb.DeleteStoryRequest) (*emptypb.Empty, error) {
	storyID, err := primitive.ObjectIDFromHex(req.StoryId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	if err := s.storyService.DeleteStory(ctx, storyID, userID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetStoriesFeed(ctx context.Context, req *storypb.GetStoriesFeedRequest) (*storypb.StoriesFeedResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	// Convert friend IDs
	friendIDs := make([]primitive.ObjectID, 0, len(req.FriendIds))
	for _, id := range req.FriendIds {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			friendIDs = append(friendIDs, oid)
		}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	stories, err := s.storyService.GetStoriesFeed(ctx, userID, friendIDs, limit, offset)
	if err != nil {
		return nil, err
	}

	protoStories := make([]*storypb.Story, 0, len(stories))
	for _, story := range stories {
		protoStories = append(protoStories, toProtoStory(&story))
	}

	return &storypb.StoriesFeedResponse{
		Stories: protoStories,
		Total:   int32(len(protoStories)),
	}, nil
}

func (s *Server) GetUserStories(ctx context.Context, req *storypb.GetUserStoriesRequest) (*storypb.StoriesResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	stories, err := s.storyService.GetUserStories(ctx, userID)
	if err != nil {
		return nil, err
	}

	protoStories := make([]*storypb.Story, 0, len(stories))
	for _, story := range stories {
		protoStories = append(protoStories, toProtoStory(&story))
	}

	return &storypb.StoriesResponse{Stories: protoStories}, nil
}

func (s *Server) RecordView(ctx context.Context, req *storypb.RecordViewRequest) (*emptypb.Empty, error) {
	storyID, err := primitive.ObjectIDFromHex(req.StoryId)
	if err != nil {
		return nil, err
	}
	viewerID, err := primitive.ObjectIDFromHex(req.ViewerId)
	if err != nil {
		return nil, err
	}

	if err := s.storyService.RecordView(ctx, storyID, viewerID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) ReactToStory(ctx context.Context, req *storypb.ReactToStoryRequest) (*emptypb.Empty, error) {
	storyID, err := primitive.ObjectIDFromHex(req.StoryId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	if err := s.storyService.ReactToStory(ctx, storyID, userID, req.ReactionType); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetStoryViewers(ctx context.Context, req *storypb.GetStoryViewersRequest) (*storypb.StoryViewersResponse, error) {
	storyID, err := primitive.ObjectIDFromHex(req.StoryId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	viewers, err := s.storyService.GetStoryViewers(ctx, storyID, userID)
	if err != nil {
		return nil, err
	}

	protoViewers := make([]*storypb.StoryViewer, 0, len(viewers))
	for _, v := range viewers {
		protoViewers = append(protoViewers, &storypb.StoryViewer{
			User: &storypb.Author{
				Id:       v.User.ID.Hex(),
				Username: v.User.Username,
				FullName: v.User.FullName,
				Avatar:   v.User.Avatar,
			},
			ReactionType: v.ReactionType,
			ViewedAt:     timestamppb.New(v.ViewedAt),
		})
	}

	return &storypb.StoryViewersResponse{Viewers: protoViewers}, nil
}

// Helper functions
func toProtoStory(s *models.Story) *storypb.Story {
	allowedViewers := make([]string, 0, len(s.AllowedViewers))
	for _, id := range s.AllowedViewers {
		allowedViewers = append(allowedViewers, id.Hex())
	}

	blockedViewers := make([]string, 0, len(s.BlockedViewers))
	for _, id := range s.BlockedViewers {
		blockedViewers = append(blockedViewers, id.Hex())
	}

	return &storypb.Story{
		Id:     s.ID.Hex(),
		UserId: s.UserID.Hex(),
		Author: &storypb.Author{
			Id:       s.Author.ID,
			Username: s.Author.Username,
			FullName: s.Author.FullName,
			Avatar:   s.Author.Avatar,
		},
		MediaUrl:       s.MediaURL,
		MediaType:      s.MediaType,
		Privacy:        string(s.Privacy),
		AllowedViewers: allowedViewers,
		BlockedViewers: blockedViewers,
		ViewCount:      int32(s.ViewCount),
		ReactionCount:  int32(s.ReactionCount),
		CreatedAt:      timestamppb.New(s.CreatedAt),
		ExpiresAt:      timestamppb.New(s.ExpiresAt),
	}
}
