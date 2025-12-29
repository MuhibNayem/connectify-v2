package graph

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jClient wraps the Neo4j driver
type Neo4jClient struct {
	Driver neo4j.DriverWithContext
}

// NewNeo4jClient creates a new Neo4j client
func NewNeo4jClient(uri, username, password string) (*Neo4jClient, error) {
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth(username, password, ""),
	)
	if err != nil {
		return nil, err
	}

	// Verify connectivity
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, err
	}

	return &Neo4jClient{Driver: driver}, nil
}

// Close closes the Neo4j driver
func (c *Neo4jClient) Close(ctx context.Context) error {
	if c.Driver != nil {
		return c.Driver.Close(ctx)
	}
	return nil
}
