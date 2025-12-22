package grpc

import (
	"context"
	"fmt"

	"gitlab.com/spydotech-group/feed-service/internal/service"
	"gitlab.com/spydotech-group/shared-entity/models"
	feedpb "gitlab.com/spydotech-group/shared-entity/proto/feed/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	feedpb.UnimplementedFeedServiceServer
	service *service.FeedService
}

func NewServer(svc *service.FeedService) *Server {
	return &Server{service: svc}
}

func (s *Server) Register(grpcServer *grpc.Server) {
	feedpb.RegisterFeedServiceServer(grpcServer, s)
}

// CreatePost implements the gRPC method
func (s *Server) CreatePost(ctx context.Context, req *feedpb.CreatePostRequest) (*feedpb.PostResponse, error) {
	post, err := s.service.CreatePost(ctx, req.UserId, req.Content, req.Privacy)
	if err != nil {
		return nil, err
	}
	return toProtoPost(post), nil
}

func (s *Server) GetPost(ctx context.Context, req *feedpb.GetPostRequest) (*feedpb.PostResponse, error) {
	post, err := s.service.GetPost(ctx, req.PostId)
	if err != nil {
		return nil, err
	}
	return toProtoPost(post), nil
}

func (s *Server) ListPosts(ctx context.Context, req *feedpb.ListPostsRequest) (*feedpb.FeedResponse, error) {
	posts, err := s.service.ListPosts(ctx, req.ViewerId, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	var protoPosts []*feedpb.PostResponse
	for _, p := range posts {
		protoPosts = append(protoPosts, toProtoPost(&p))
	}

	return &feedpb.FeedResponse{
		Posts: protoPosts,
		Total: int64(len(posts)), // Placeholder
		Page:  req.Page,
		Limit: req.Limit,
	}, nil
}

// Helper to convert Model -> Proto
func toProtoPost(p *models.Post) *feedpb.PostResponse {
	return &feedpb.PostResponse{
		Id:      p.ID.Hex(),
		UserId:  p.UserID.Hex(),
		Content: p.Content,
		Privacy: string(p.Privacy),
		// ... Map other fields
	}
}

// Unimplemented methods...
func (s *Server) UpdatePost(context.Context, *feedpb.UpdatePostRequest) (*feedpb.PostResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (s *Server) DeletePost(context.Context, *feedpb.DeletePostRequest) (*emptypb.Empty, error) {
	return nil, fmt.Errorf("unimplemented")
}

// ...
