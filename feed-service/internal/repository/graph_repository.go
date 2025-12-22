package repository

import (
	"context"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GraphRepository struct {
	driver neo4j.DriverWithContext
}

func NewGraphRepository(driver neo4j.DriverWithContext) *GraphRepository {
	return &GraphRepository{driver: driver}
}

// SyncUser ensures a user node exists
func (r *GraphRepository) SyncUser(ctx context.Context, userID string) error {
	query := `MERGE (u:User {id: $userID})`
	params := map[string]any{"userID": userID}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err == nil {
		log.Printf("[Neo4j] Synced User %s", userID)
	}
	return err
}

// UpdateFriendship handles ACCEPTED, REMOVED, and BLOCKED on the graph
func (r *GraphRepository) UpdateFriendship(ctx context.Context, requesterID, receiverID, status string) error {
	var query string
	params := map[string]any{
		"u1": requesterID,
		"u2": receiverID,
	}

	switch status {
	case "accepted":
		// Create FRIEND relationship, remove any pending requests if we were tracking them
		query = `
			MERGE (u1:User {id: $u1})
			MERGE (u2:User {id: $u2})
			MERGE (u1)-[f:FRIEND]-(u2) // Undirected/Bi-directional semantics
			ON CREATE SET f.since = datetime()
		`
	case "removed":
		// Remove FRIEND relationship
		query = `
			MATCH (u1:User {id: $u1})-[r:FRIEND]-(u2:User {id: $u2})
			DELETE r
		`
	case "blocked":
		// Remove FRIEND, Create BLOCKED (assuming directional block from u1 -> u2? Event should specify blocker)
		// For simplicity, we just break the friend link here as feed service cares about visibility.
		query = `
			MATCH (u1:User {id: $u1})-[r:FRIEND]-(u2:User {id: $u2})
			DELETE r
		`
	default:
		return nil
	}

	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

// GetFriendIDs using Graph
func (r *GraphRepository) GetFriendIDs(ctx context.Context, userID primitive.ObjectID) ([]string, error) {
	query := `MATCH (u:User {id: $userID})-[:FRIEND]-(f:User) RETURN f.id`
	params := map[string]any{"userID": userID.Hex()}

	result, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, rec := range result.Records {
		if id, ok := rec.Values[0].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
