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
