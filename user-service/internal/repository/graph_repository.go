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

func (r *GraphRepository) SyncUser(ctx context.Context, userID primitive.ObjectID) error {
	query := `MERGE (u:User {id: $userID})`
	params := map[string]any{"userID": userID.Hex()}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err == nil {
		log.Printf("[Neo4j] Synced User %s", userID.Hex())
	}
	return err
}

func (r *GraphRepository) SendRequest(ctx context.Context, from, to primitive.ObjectID) error {
	query := `
		MERGE (u1:User {id: $from})
		MERGE (u2:User {id: $to})
		MERGE (u1)-[r:REQUESTED]->(u2)
		ON CREATE SET r.created_at = datetime()
	`
	params := map[string]any{"from": from.Hex(), "to": to.Hex()}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

func (r *GraphRepository) AcceptRequest(ctx context.Context, from, to primitive.ObjectID) error {
	query := `
		MATCH (u1:User {id: $from})
		MATCH (u2:User {id: $to})
		OPTIONAL MATCH (u1)-[r:REQUESTED]-(u2)
		DELETE r
		MERGE (u1)-[f:FRIEND]-(u2)
		ON CREATE SET f.since = datetime()
	`
	params := map[string]any{"from": from.Hex(), "to": to.Hex()}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

func (r *GraphRepository) RejectRequest(ctx context.Context, from, to primitive.ObjectID) error {
	query := `MATCH (u1:User {id: $from})-[r:REQUESTED]-(u2:User {id: $to}) DELETE r`
	params := map[string]any{"from": from.Hex(), "to": to.Hex()}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

func (r *GraphRepository) Unfriend(ctx context.Context, user1, user2 primitive.ObjectID) error {
	query := `MATCH (u1:User {id: $user1})-[r:FRIEND]-(u2:User {id: $user2}) DELETE r`
	params := map[string]any{"user1": user1.Hex(), "user2": user2.Hex()}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

func (r *GraphRepository) BlockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	query := `
		MERGE (u1:User {id: $blocker})
		MERGE (u2:User {id: $blocked})
		MERGE (u1)-[b:BLOCKED]->(u2)
		ON CREATE SET b.created_at = datetime()
		WITH u1, u2
		OPTIONAL MATCH (u1)-[r:FRIEND|REQUESTED]-(u2)
		DELETE r
	`
	params := map[string]any{"blocker": blocker.Hex(), "blocked": blocked.Hex()}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

func (r *GraphRepository) UnblockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	query := `MATCH (u1:User {id: $blocker})-[r:BLOCKED]->(u2:User {id: $blocked}) DELETE r`
	params := map[string]any{"blocker": blocker.Hex(), "blocked": blocked.Hex()}
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

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
