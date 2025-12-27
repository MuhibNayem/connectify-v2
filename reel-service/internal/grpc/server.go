package reelgrpc

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/reel-service/internal/service"
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
	CreateReel(ctx context.Context, userID primitive.ObjectID, req service.CreateReelRequest) (*models.Reel, error)
	DeleteReel(ctx context.Context, reelID, userID primitive.ObjectID) error
	AddComment(ctx context.Context, reelID, userID primitive.ObjectID, content string, explicitMentions []primitive.ObjectID) (*models.Comment, error)
	GetComments(ctx context.Context, reelID primitive.ObjectID, limit, offset int64) ([]models.Comment, error)
	AddReply(ctx context.Context, reelID, commentID, userID primitive.ObjectID, content string) (*models.Reply, error)
	ReactToComment(ctx context.Context, reelID, commentID, userID primitive.ObjectID, reactionType models.ReactionType) error
	ReactToReel(ctx context.Context, reelID, userID primitive.ObjectID, reactionType models.ReactionType) error
	IncrementViews(ctx context.Context, reelID, viewerID primitive.ObjectID) error
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

func (s *Server) CreateReel(ctx context.Context, req *reelpb.CreateReelRequest) (*reelpb.CreateReelResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}

	allowedViewers := toObjectIDs(req.AllowedViewers)
	blockedViewers := toObjectIDs(req.BlockedViewers)

	// Context validation would happen here or in service
	// We call the service which now handles author resolution internally
	serviceReq := service.CreateReelRequest{
		VideoURL:       req.VideoUrl,
		ThumbnailURL:   req.ThumbnailUrl,
		Caption:        req.Caption,
		Duration:       int(req.Duration),
		Privacy:        models.PrivacySettingType(req.Privacy),
		AllowedViewers: allowedViewers,
		BlockedViewers: blockedViewers,
	}

	reel, err := s.svc.CreateReel(ctx, userID, serviceReq)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &reelpb.CreateReelResponse{
		Reel: toProtoReel(reel),
	}, nil
}

func (s *Server) DeleteReel(ctx context.Context, req *reelpb.DeleteReelRequest) (*reelpb.DeleteReelResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}

	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}

	if err := s.svc.DeleteReel(ctx, reelID, userID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &reelpb.DeleteReelResponse{Success: true}, nil
}

func (s *Server) GetComments(ctx context.Context, req *reelpb.GetCommentsRequest) (*reelpb.GetCommentsResponse, error) {
	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}

	comments, err := s.svc.GetComments(ctx, reelID, req.Limit, req.Offset)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoComments := make([]*reelpb.Comment, len(comments))
	for i, c := range comments {
		protoComments[i] = toProtoComment(&c)
	}

	return &reelpb.GetCommentsResponse{
		Comments: protoComments,
	}, nil
}

func (s *Server) AddComment(ctx context.Context, req *reelpb.AddCommentRequest) (*reelpb.AddCommentResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}
	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}

	mentions := toObjectIDs(req.ExplicitMentions)

	comment, err := s.svc.AddComment(ctx, reelID, userID, req.Content, mentions)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &reelpb.AddCommentResponse{Comment: toProtoComment(comment)}, nil
}

func (s *Server) AddReply(ctx context.Context, req *reelpb.AddReplyRequest) (*reelpb.AddReplyResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}
	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}
	commentID, err := primitive.ObjectIDFromHex(req.CommentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid comment ID")
	}

	reply, err := s.svc.AddReply(ctx, reelID, commentID, userID, req.Content)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &reelpb.AddReplyResponse{Reply: toProtoReply(reply)}, nil
}

func (s *Server) ReactToComment(ctx context.Context, req *reelpb.ReactToCommentRequest) (*reelpb.ReactToCommentResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}
	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}
	commentID, err := primitive.ObjectIDFromHex(req.CommentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid comment ID")
	}

	if err := s.svc.ReactToComment(ctx, reelID, commentID, userID, models.ReactionType(req.ReactionType)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &reelpb.ReactToCommentResponse{Success: true}, nil
}

func (s *Server) ReactToReel(ctx context.Context, req *reelpb.ReactToReelRequest) (*reelpb.ReactToReelResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}
	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}

	if err := s.svc.ReactToReel(ctx, reelID, userID, models.ReactionType(req.ReactionType)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &reelpb.ReactToReelResponse{Success: true}, nil
}

func (s *Server) IncrementView(ctx context.Context, req *reelpb.IncrementViewRequest) (*reelpb.IncrementViewResponse, error) {
	var viewerID primitive.ObjectID
	var err error

	if req.ViewerId != "" {
		viewerID, err = primitive.ObjectIDFromHex(req.ViewerId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid viewer ID")
		}
	} else {
		viewerID = primitive.NilObjectID
	}
	reelID, err := primitive.ObjectIDFromHex(req.ReelId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid reel ID")
	}

	if err := s.svc.IncrementViews(ctx, reelID, viewerID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &reelpb.IncrementViewResponse{Success: true}, nil
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

func toProtoComment(c *models.Comment) *reelpb.Comment {
	replies := make([]*reelpb.Reply, len(c.Replies))
	for i, r := range c.Replies {
		replies[i] = toProtoReply(&r)
	}

	likeCount := int64(0)
	if c.ReactionCounts != nil {
		likeCount = c.ReactionCounts[models.ReactionLike]
	}

	return &reelpb.Comment{
		Id:     c.ID.Hex(),
		UserId: c.UserID.Hex(),
		Author: &reelpb.Author{
			Id:       c.Author.ID,
			Username: c.Author.Username,
			Avatar:   c.Author.Avatar,
			FullName: c.Author.FullName,
		},
		Content:    c.Content,
		Mentions:   toStrings(c.Mentions),
		LikeCount:  likeCount,
		ReplyCount: int64(len(c.Replies)),
		Replies:    replies,
		CreatedAt:  timestamppb.New(c.CreatedAt),
	}
}

func toProtoReply(r *models.Reply) *reelpb.Reply {
	likeCount := int64(0)
	if r.ReactionCounts != nil {
		likeCount = r.ReactionCounts[models.ReactionLike]
	}

	return &reelpb.Reply{
		Id:        r.ID.Hex(),
		CommentId: r.CommentID.Hex(),
		UserId:    r.UserID.Hex(),
		Author: &reelpb.Author{
			Id:       r.Author.ID,
			Username: r.Author.Username,
			Avatar:   r.Author.Avatar,
			FullName: r.Author.FullName,
		},
		Content:   r.Content,
		Mentions:  toStrings(r.Mentions),
		LikeCount: likeCount,
		CreatedAt: timestamppb.New(r.CreatedAt),
	}
}

func toObjectIDs(ids []string) []primitive.ObjectID {
	oids := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			oids = append(oids, oid)
		}
	}
	return oids
}

func toStrings(oids []primitive.ObjectID) []string {
	ids := make([]string, len(oids))
	for i, oid := range oids {
		ids[i] = oid.Hex()
	}
	return ids
}
