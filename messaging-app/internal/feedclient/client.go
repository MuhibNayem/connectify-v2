package feedclient

import (
	"context"
	"fmt"
	"time"

	"messaging-app/config"

	feedpb "gitlab.com/spydotech-group/shared-entity/proto/feed/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn    *grpc.ClientConn
	service feedpb.FeedServiceClient
}

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	// Assuming configuration for Feed Service is available
	// Defaulting to localhost:9098 if not explicitly set
	addr := "localhost:9098"

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("feed service connection failed: %w", err)
	}

	return &Client{
		conn:    conn,
		service: feedpb.NewFeedServiceClient(conn),
	}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

// Proxy methods to gRPC

func (c *Client) CreatePost(ctx context.Context, req *feedpb.CreatePostRequest) (*feedpb.PostResponse, error) {
	return c.service.CreatePost(ctx, req)
}

func (c *Client) GetPost(ctx context.Context, req *feedpb.GetPostRequest) (*feedpb.PostResponse, error) {
	return c.service.GetPost(ctx, req)
}

func (c *Client) ListPosts(ctx context.Context, req *feedpb.ListPostsRequest) (*feedpb.FeedResponse, error) {
	return c.service.ListPosts(ctx, req)
}
