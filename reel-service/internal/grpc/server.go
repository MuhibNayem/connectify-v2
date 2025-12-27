package reelgrpc

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	reelpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/reel/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReelService interface {
	GetReel(ctx context.Context, reelID primitive.ObjectID) (*models.Reel, error)
	GetUserReels(ctx context.Context, userID primitive.ObjectID) ([]models.Reel, error)
	GetReelsFeed(ctx context.Context, viewerID primitive.ObjectID, limit, offset int64) ([]models.Reel, error)
}

type Server struct {
	reelpb.UnimplementedReelServiceServer
	svc ReelService
}

func NewServer(svc ReelService) *Server {
	return &Server{svc: svc}
}

func (s *Server) Register(grpcServer *grpc.Server) {
	reelpb.RegisterReelServiceServer(grpcServer, s)
}

func (s *Server) GetReel(ctx context.Context, req *reelpb.GetReelRequest) (*reelpb.GetReelResponse, error) {
	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}

	reel, err := s.svc.GetReel(ctx, reelID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Reel not found")
	}

	return &reelpb.GetReelResponse{
		Reel: toProtoReel(reel),
	}, nil
}

func (s *Server) GetUserReels(ctx context.Context, req *reelpb.GetUserReelsRequest) (*reelpb.GetUserReelsResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}

	reels, err := s.svc.GetUserReels(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoReels := make([]*reelpb.Reel, len(reels))
	for i, r := range reels {
		protoReels[i] = toProtoReel(&r)
	}

	return &reelpb.GetUserReelsResponse{
		Reels: protoReels,
	}, nil
}

func (s *Server) GetReelsFeed(ctx context.Context, req *reelpb.GetReelsFeedRequest) (*reelpb.GetReelsFeedResponse, error) {
	viewerID, err := primitive.ObjectIDFromHex(req.ViewerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid viewer ID")
	}

	reels, err := s.svc.GetReelsFeed(ctx, viewerID, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoReels := make([]*reelpb.Reel, len(reels))
	for i, r := range reels {
		protoReels[i] = toProtoReel(&r)
	}

	return &reelpb.GetReelsFeedResponse{
		Reels: protoReels,
	}, nil
}

func toProtoReel(r *models.Reel) *reelpb.Reel {
	return &reelpb.Reel{
		Id:           r.ID.Hex(),
		UserId:       r.UserID.Hex(),
		VideoUrl:     r.VideoURL,
		ThumbnailUrl: r.ThumbnailURL,
		Caption:      r.Caption,
		Duration:     float64(r.Duration),
		Privacy:      string(r.Privacy),
		Views:        r.Views,
		Likes:        r.Likes,
		Comments:     r.Comments,
		Author: &reelpb.Author{
			Id:       r.Author.ID,
			Username: r.Author.Username,
			Avatar:   r.Author.Avatar,
			FullName: r.Author.FullName,
		},
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}
