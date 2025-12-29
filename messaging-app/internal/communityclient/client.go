package communityclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"messaging-app/config"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	communitypb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/community/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client communitypb.CommunityServiceClient
	cb     *resilience.CircuitBreaker
}

func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	addr := net.JoinHostPort(cfg.CommunityGRPCHost, cfg.CommunityGRPCPort)
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
		return nil, fmt.Errorf("connect to community gRPC at %s: %w", addr, err)
	}

	cbConfig := resilience.DefaultConfig("community-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: communitypb.NewCommunityServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
