package reelclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"messaging-app/config"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	reelpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/reel/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client reelpb.ReelServiceClient
	cb     *resilience.CircuitBreaker
}

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	addr := net.JoinHostPort(cfg.ReelGRPCHost, cfg.ReelGRPCPort)
	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		observability.GetGRPCDialOption(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to reel gRPC at %s: %w", addr, err)
	}

	cbConfig := resilience.DefaultConfig("reel-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: reelpb.NewReelServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) GetClient() reelpb.ReelServiceClient {
	return c.client
}

func (c *Client) CircuitBreaker() *resilience.CircuitBreaker {
	return c.cb
}

func (c *Client) CreateReel(ctx context.Context, req *reelpb.CreateReelRequest) (*reelpb.Reel, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.CreateReel(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("create reel: %w", err)
	}
	return result.(*reelpb.CreateReelResponse).Reel, nil
}

func (c *Client) GetReel(ctx context.Context, reelID string) (*reelpb.Reel, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetReel(ctx, &reelpb.GetReelRequest{ReelId: reelID})
	})
	if err != nil {
		return nil, fmt.Errorf("get reel %s: %w", reelID, err)
	}
	return result.(*reelpb.GetReelResponse).Reel, nil
}

func (c *Client) GetUserReels(ctx context.Context, userID string) ([]*reelpb.Reel, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetUserReels(ctx, &reelpb.GetUserReelsRequest{UserId: userID})
	})
	if err != nil {
		return nil, fmt.Errorf("get user reels %s: %w", userID, err)
	}
	return result.(*reelpb.GetUserReelsResponse).Reels, nil
}

func (c *Client) DeleteReel(ctx context.Context, reelID, userID string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DeleteReel(ctx, &reelpb.DeleteReelRequest{ReelId: reelID, UserId: userID})
	})
	if err != nil {
		return fmt.Errorf("delete reel %s: %w", reelID, err)
	}
	return nil
}

func (c *Client) GetReelsFeed(ctx context.Context, viewerID string, limit, offset int64) ([]*reelpb.Reel, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetReelsFeed(ctx, &reelpb.GetReelsFeedRequest{
			ViewerId: viewerID,
			Limit:    limit,
			Offset:   offset,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("get reels feed: %w", err)
	}
	return result.(*reelpb.GetReelsFeedResponse).Reels, nil
}

func (c *Client) IncrementView(ctx context.Context, reelID, viewerID string) error {
	// Not critical, maybe skip circuit breaker? Or keep it.
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.IncrementView(ctx, &reelpb.IncrementViewRequest{ReelId: reelID, ViewerId: viewerID})
	})
	return err
}

func (c *Client) ReactToReel(ctx context.Context, reelID, userID string, reactionType string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ReactToReel(ctx, &reelpb.ReactToReelRequest{
			ReelId:       reelID,
			UserId:       userID,
			ReactionType: reactionType,
		})
	})
	return err
}

func (c *Client) AddComment(ctx context.Context, req *reelpb.AddCommentRequest) (*reelpb.Comment, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.AddComment(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("add comment: %w", err)
	}
	return result.(*reelpb.AddCommentResponse).Comment, nil
}

func (c *Client) GetComments(ctx context.Context, reelID string, limit, offset int64) ([]*reelpb.Comment, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetComments(ctx, &reelpb.GetCommentsRequest{
			ReelId: reelID,
			Limit:  limit,
			Offset: offset,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}
	return result.(*reelpb.GetCommentsResponse).Comments, nil
}

func (c *Client) AddReply(ctx context.Context, req *reelpb.AddReplyRequest) (*reelpb.Reply, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.AddReply(ctx, req)
	})
	if err != nil {
		return nil, fmt.Errorf("add reply: %w", err)
	}
	return result.(*reelpb.AddReplyResponse).Reply, nil
}

func (c *Client) ReactToComment(ctx context.Context, reelID, commentID, userID string, reactionType string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ReactToComment(ctx, &reelpb.ReactToCommentRequest{
			ReelId:       reelID,
			CommentId:    commentID,
			UserId:       userID,
			ReactionType: reactionType,
		})
	})
	return err
}
