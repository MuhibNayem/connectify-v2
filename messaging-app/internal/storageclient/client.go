package storageclient

import (
	"context"
	"fmt"
	"log"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	storagepb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/storage/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client storagepb.StorageServiceClient
	cb     *resilience.CircuitBreaker
}

func NewClient(host, port string) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		observability.GetGRPCDialOption(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to storage-service: %w", err)
	}

	log.Printf("Connected to storage-service at %s", addr)

	cbConfig := resilience.DefaultConfig("storage-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: storagepb.NewStorageServiceClient(conn),
		cb:     cb,
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) GetClient() storagepb.StorageServiceClient {
	return c.client
}

func (c *Client) Ping(ctx context.Context) error {
	return nil
}
