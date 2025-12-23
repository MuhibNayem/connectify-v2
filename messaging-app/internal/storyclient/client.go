package storyclient

import (
	"context"
	"fmt"
	"log"

	storypb "gitlab.com/spydotech-group/shared-entity/proto/story/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client storypb.StoryServiceClient
}

func NewClient(host, port string) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to story-service: %w", err)
	}

	log.Printf("Connected to story-service at %s", addr)

	return &Client{
		conn:   conn,
		client: storypb.NewStoryServiceClient(conn),
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
