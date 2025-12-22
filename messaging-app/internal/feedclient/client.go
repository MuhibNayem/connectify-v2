package feedclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"messaging-app/config"

	feedpb "gitlab.com/spydotech-group/shared-entity/proto/feed/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client feedpb.FeedServiceClient
}

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	addr := net.JoinHostPort(cfg.FeedServiceHost, cfg.FeedServicePort)

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("feed service connection failed: %w", err)
	}

	return &Client{
		conn:   conn,
		client: feedpb.NewFeedServiceClient(conn),
	}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}
