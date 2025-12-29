package friendshipclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"messaging-app/config"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	friendshippb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/friendship/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC connection to the Friendship service
type Client struct {
	conn   *grpc.ClientConn
	client friendshippb.FriendshipServiceClient
	cb     *resilience.CircuitBreaker
}

// New creates a new Friendship gRPC client using the configured host/port
func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	addr := net.JoinHostPort(cfg.FriendshipGRPCHost, cfg.FriendshipGRPCPort)
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
		return nil, fmt.Errorf("connect to friendship gRPC at %s: %w", addr, err)
	}

	// Create circuit breaker with default config
	cbConfig := resilience.DefaultConfig("friendship-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: friendshippb.NewFriendshipServiceClient(conn),
		cb:     cb,
	}, nil
}

// NewClient creates a client with an address string (for backward compatibility)
func NewClient(address string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		observability.GetGRPCDialOption(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to friendship gRPC at %s: %w", address, err)
	}

	cbConfig := resilience.DefaultConfig("friendship-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: friendshippb.NewFriendshipServiceClient(conn),
		cb:     cb,
	}, nil
}

// Close shuts down the underlying gRPC connection
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Service returns the underlying gRPC client for direct access if needed
func (c *Client) Service() friendshippb.FriendshipServiceClient {
	return c.client
}
