package feedclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"messaging-app/config"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	feedpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/feed/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client feedpb.FeedServiceClient
	cb     *resilience.CircuitBreaker
}

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	addr := net.JoinHostPort(cfg.FeedServiceHost, cfg.FeedServicePort)

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		observability.GetGRPCDialOption(),
	)
	if err != nil {
		return nil, fmt.Errorf("feed service connection failed: %w", err)
	}

	// Create circuit breaker with default config
	cbConfig := resilience.DefaultConfig("feed-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: feedpb.NewFeedServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}
