package grpc

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/feed-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	feedpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/feed/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (s *Server) UpdatePost(ctx context.Context, req *feedpb.UpdatePostRequest) (*feedpb.PostResponse, error) {
	post, err := s.service.UpdatePost(ctx, req.PostId, req.UserId, req.Content, req.Privacy)
	if err != nil {
		return nil, err
	}
	return toProtoPost(post), nil
}

func (s *Server) DeletePost(ctx context.Context, req *feedpb.DeletePostRequest) (*emptypb.Empty, error) {
	err := s.service.DeletePost(ctx, req.PostId, req.UserId)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
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

func (s *Server) GetPostsByHashtag(ctx context.Context, req *feedpb.GetPostsByHashtagRequest) (*feedpb.FeedResponse, error) {
	posts, err := s.service.GetPostsByHashtag(ctx, req.ViewerId, req.Hashtag, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	var protoPosts []*feedpb.PostResponse
	for _, p := range posts {
		protoPosts = append(protoPosts, toProtoPost(&p))
	}

	return &feedpb.FeedResponse{
		Posts: protoPosts,
		Total: int64(len(posts)),
		Page:  req.Page,
		Limit: req.Limit,
	}, nil
}

func (s *Server) ReactToPost(ctx context.Context, req *feedpb.ReactToPostRequest) (*emptypb.Empty, error) {
	err := s.service.ReactToPost(ctx, req.UserId, req.PostId, req.Emoji)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) ReactToComment(ctx context.Context, req *feedpb.ReactToCommentRequest) (*emptypb.Empty, error) {
	err := s.service.ReactToComment(ctx, req.UserId, req.CommentId, req.Emoji)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) ReactToReply(ctx context.Context, req *feedpb.ReactToReplyRequest) (*emptypb.Empty, error) {
	err := s.service.ReactToReply(ctx, req.UserId, req.ReplyId, req.Emoji)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Helper to convert Model -> Proto
func toProtoPost(p *models.Post) *feedpb.PostResponse {
	return &feedpb.PostResponse{
		Id:             p.ID.Hex(),
		UserId:         p.UserID.Hex(),
		Content:        p.Content,
		Privacy:        string(p.Privacy),
		TotalReactions: p.TotalReactions,
		TotalComments:  p.TotalComments,
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
	}
}

func toProtoComment(c *models.Comment) *feedpb.CommentResponse {
	return &feedpb.CommentResponse{
		Id:        c.ID.Hex(),
		PostId:    c.PostID.Hex(),
		UserId:    c.UserID.Hex(),
		Content:   c.Content,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func toProtoReply(r *models.Reply) *feedpb.ReplyResponse {
	return &feedpb.ReplyResponse{
		Id:        r.ID.Hex(),
		CommentId: r.CommentID.Hex(),
		UserId:    r.UserID.Hex(),
		Content:   r.Content,
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

// ----------------------------- Albums -----------------------------

func (s *Server) CreateAlbum(ctx context.Context, req *feedpb.CreateAlbumRequest) (*feedpb.AlbumResponse, error) {
	album, err := s.service.CreateAlbum(ctx, req.UserId, req.Title, req.Description, req.Privacy)
	if err != nil {
		return nil, err
	}
	return toProtoAlbum(album), nil
}

func (s *Server) GetAlbum(ctx context.Context, req *feedpb.GetAlbumRequest) (*feedpb.AlbumResponse, error) {
	album, err := s.service.GetAlbum(ctx, req.AlbumId)
	if err != nil {
		return nil, err
	}
	return toProtoAlbum(album), nil
}

func (s *Server) UpdateAlbum(ctx context.Context, req *feedpb.UpdateAlbumRequest) (*feedpb.AlbumResponse, error) {
	album, err := s.service.UpdateAlbum(ctx, req.AlbumId, req.Title, req.Description, req.Privacy)
	if err != nil {
		return nil, err
	}
	return toProtoAlbum(album), nil
}

func (s *Server) DeleteAlbum(ctx context.Context, req *feedpb.DeleteAlbumRequest) (*emptypb.Empty, error) {
	err := s.service.DeleteAlbum(ctx, req.UserId, req.AlbumId)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) ListAlbums(ctx context.Context, req *feedpb.ListAlbumsRequest) (*feedpb.ListAlbumsResponse, error) {
	albums, err := s.service.ListAlbums(ctx, req.UserId, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	var protoAlbums []*feedpb.AlbumResponse
	for _, a := range albums {
		protoAlbums = append(protoAlbums, toProtoAlbum(&a))
	}

	return &feedpb.ListAlbumsResponse{
		Albums: protoAlbums,
	}, nil
}

// ----------------------------- Album Media -----------------------------

func (s *Server) AddMediaToAlbum(ctx context.Context, req *feedpb.AddMediaToAlbumRequest) (*feedpb.AlbumMediaResponse, error) {
	media, err := s.service.AddMediaToAlbum(ctx, req.AlbumId, req.Url, req.Type, req.Description)
	if err != nil {
		return nil, err
	}
	return toProtoAlbumMedia(media), nil
}

func (s *Server) RemoveMediaFromAlbum(ctx context.Context, req *feedpb.RemoveMediaFromAlbumRequest) (*emptypb.Empty, error) {
	err := s.service.RemoveMediaFromAlbum(ctx, req.AlbumId, req.MediaId)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetAlbumMedia(ctx context.Context, req *feedpb.GetAlbumMediaRequest) (*feedpb.GetAlbumMediaResponse, error) {
	mediaList, err := s.service.GetAlbumMedia(ctx, req.AlbumId, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	var protoMedia []*feedpb.AlbumMediaResponse
	for _, m := range mediaList {
		protoMedia = append(protoMedia, toProtoAlbumMedia(&m))
	}

	return &feedpb.GetAlbumMediaResponse{
		Media: protoMedia,
	}, nil
}

// Helpers

func toProtoAlbum(a *models.Album) *feedpb.AlbumResponse {
	return &feedpb.AlbumResponse{
		Id:          a.ID.Hex(),
		UserId:      a.UserID.Hex(),
		Title:       a.Title,
		Description: a.Description,
		Privacy:     string(a.Privacy),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
	}
}

func toProtoAlbumMedia(m *models.AlbumMedia) *feedpb.AlbumMediaResponse {
	return &feedpb.AlbumMediaResponse{
		Id:          m.ID.Hex(),
		AlbumId:     m.AlbumID.Hex(),
		Url:         m.URL,
		Type:        m.Type,
		Description: m.Description,
		CreatedAt:   timestamppb.New(m.CreatedAt),
	}
}

func (s *Server) UpdatePostStatus(ctx context.Context, req *feedpb.UpdatePostStatusRequest) (*emptypb.Empty, error) {
	err := s.service.UpdatePostStatus(ctx, req.PostId, req.UserId, req.Status)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Unimplemented methods...
func (s *Server) CreateComment(ctx context.Context, req *feedpb.CreateCommentRequest) (*feedpb.CommentResponse, error) {
	comment, err := s.service.CreateComment(ctx, req.UserId, req.PostId, req.Content)
	if err != nil {
		return nil, err
	}
	return toProtoComment(comment), nil
}

func (s *Server) ListComments(ctx context.Context, req *feedpb.ListCommentsRequest) (*feedpb.ListCommentsResponse, error) {
	comments, err := s.service.ListComments(ctx, req.PostId, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}
	var protoComments []*feedpb.CommentResponse
	for _, c := range comments {
		protoComments = append(protoComments, toProtoComment(&c))
	}
	return &feedpb.ListCommentsResponse{Comments: protoComments}, nil
}

func (s *Server) CreateReply(ctx context.Context, req *feedpb.CreateReplyRequest) (*feedpb.ReplyResponse, error) {
	reply, err := s.service.CreateReply(ctx, req.UserId, req.CommentId, req.Content)
	if err != nil {
		return nil, err
	}
	return toProtoReply(reply), nil
}

func (s *Server) ListReplies(ctx context.Context, req *feedpb.ListRepliesRequest) (*feedpb.ListRepliesResponse, error) {
	replies, err := s.service.ListReplies(ctx, req.CommentId, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}
	var protoReplies []*feedpb.ReplyResponse
	for _, r := range replies {
		protoReplies = append(protoReplies, toProtoReply(&r))
	}
	return &feedpb.ListRepliesResponse{Replies: protoReplies}, nil
}

// ...
