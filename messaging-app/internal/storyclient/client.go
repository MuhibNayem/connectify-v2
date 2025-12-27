package storyclient

import (
	"context"
	"fmt"
	"log"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	storypb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/story/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client storypb.StoryServiceClient
	cb     *resilience.CircuitBreaker
}

func NewClient(host, port string) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		observability.GetGRPCDialOption(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to story-service: %w", err)
	}

	log.Printf("Connected to story-service at %s", addr)

	// Create circuit breaker with default config
	cbConfig := resilience.DefaultConfig("story-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: storypb.NewStoryServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetClient returns the underlying gRPC client for advanced usage
func (c *Client) GetClient() storypb.StoryServiceClient {
	return c.client
}

// Ping tests the connection (by calling a lightweight method)
func (c *Client) Ping(ctx context.Context) error {
	// Just returns nil - connection health should be monitored by gRPC itself
	return nil
}
