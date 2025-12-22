package userclient

import (
	"context"
	"fmt"
	"time"

	"messaging-app/config"

	pb "gitlab.com/spydotech-group/shared-entity/proto/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC connection to the User service
type Client struct {
	conn   *grpc.ClientConn
	client pb.UserServiceClient
}

// New creates a new User gRPC client using the configured host/port
func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	// Parse URL or use Host/Port. The config currently has UserServiceURL (full string)
	// We might need to adjust or parse it.
	// For now, let's assume UserServiceURL is "host:port".
	// If the pattern uses Host/Port separate fields, we should check config.

	// Check config pattern used in others:
	// cfg.EventsGRPCHost, cfg.EventsGRPCPort

	// Reviewing config.go later to see if we have separate fields.
	// If only UserServiceURL exists, we use it directly.

	addr := cfg.UserServiceURL
	if addr == "" {
		return nil, fmt.Errorf("user service URL is empty")
	}

	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to user gRPC at %s: %w", addr, err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewUserServiceClient(conn),
	}, nil
}

// Close shuts down the underlying gRPC connection
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
