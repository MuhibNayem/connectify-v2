package userclient

import (
	"context"
	"fmt"
	"net"
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
	addr := net.JoinHostPort(cfg.UserServiceHost, cfg.UserServicePort)

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
