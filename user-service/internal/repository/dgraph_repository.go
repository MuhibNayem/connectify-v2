package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DgraphRepository implements GraphClient using Dgraph
type DgraphRepository struct {
	client *dgo.Dgraph
}

// NewDgraphRepository creates a new Dgraph repository
func NewDgraphRepository(addr string) (*DgraphRepository, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to dgraph: %w", err)
	}

	dc := api.NewDgraphClient(conn)
	dgraphClient := dgo.NewDgraphClient(dc)

	repo := &DgraphRepository{client: dgraphClient}

	// Initialize schema in background
	go func() {
		if err := repo.setupSchema(); err != nil {
			slog.Error("Failed to setup Dgraph schema for user-service", "error", err)
		}
	}()

	return repo, nil
}

func (r *DgraphRepository) setupSchema() error {
	schema := `
		type User {
			userID
			friends
			requested
			blocked
		}
		
		userID: string @index(exact) @upsert .
		friends: [uid] @reverse .
		requested: [uid] @reverse .
		blocked: [uid] @reverse .
		created_at: datetime .
		since: datetime .
	`
	return r.client.Alter(context.Background(), &api.Operation{Schema: schema})
}

// Ensure implementation
var _ GraphClient = (*DgraphRepository)(nil)

// SyncUser ensures a user exists in the graph
func (r *DgraphRepository) SyncUser(ctx context.Context, userID primitive.ObjectID) error {
	q := `query {
		u as var(func: eq(userID, "` + userID.Hex() + `"))
	}`

	mu := &api.Mutation{
		SetNquads: []byte(`uid(u) <userID> "` + userID.Hex() + `" .
			uid(u) <dgraph.type> "User" .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// SendRequest creates a friend request relationship
func (r *DgraphRepository) SendRequest(ctx context.Context, from, to primitive.ObjectID) error {
	q := `query {
		u1 as var(func: eq(userID, "` + from.Hex() + `"))
		u2 as var(func: eq(userID, "` + to.Hex() + `"))
	}`

	mu := &api.Mutation{
		SetNquads: []byte(`uid(u1) <requested> uid(u2) .
			uid(u1) <userID> "` + from.Hex() + `" .
			uid(u1) <dgraph.type> "User" .
			uid(u2) <userID> "` + to.Hex() + `" .
			uid(u2) <dgraph.type> "User" .
			uid(u1) <created_at> "` + r.now() + `" (since=request) .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// AcceptRequest transitions a request to a friendship
func (r *DgraphRepository) AcceptRequest(ctx context.Context, from, to primitive.ObjectID) error {
	q := `query {
		u1 as var(func: eq(userID, "` + from.Hex() + `"))
		u2 as var(func: eq(userID, "` + to.Hex() + `"))
	}`

	// Remove request
	dels := `uid(u1) <requested> uid(u2) .
		uid(u2) <requested> uid(u1) .`

	// Add friends (bidirectional)
	sets := `uid(u1) <friends> uid(u2) .
		uid(u2) <friends> uid(u1) .
		uid(u1) <since> "` + r.now() + `" .`

	mu := &api.Mutation{
		DelNquads: []byte(dels),
		SetNquads: []byte(sets),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// RejectRequest removes a friend request
func (r *DgraphRepository) RejectRequest(ctx context.Context, from, to primitive.ObjectID) error {
	q := `query {
		u1 as var(func: eq(userID, "` + from.Hex() + `"))
		u2 as var(func: eq(userID, "` + to.Hex() + `"))
	}`

	dels := `uid(u1) <requested> uid(u2) .
		uid(u2) <requested> uid(u1) .`

	mu := &api.Mutation{
		DelNquads: []byte(dels),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// Unfriend removes a friendship relationship
func (r *DgraphRepository) Unfriend(ctx context.Context, user1, user2 primitive.ObjectID) error {
	q := `query {
		u1 as var(func: eq(userID, "` + user1.Hex() + `"))
		u2 as var(func: eq(userID, "` + user2.Hex() + `"))
	}`

	dels := `uid(u1) <friends> uid(u2) .
		uid(u2) <friends> uid(u1) .`

	mu := &api.Mutation{
		DelNquads: []byte(dels),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// BlockUser creates a block relationship and removes any existing friendship/request
func (r *DgraphRepository) BlockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	q := `query {
		u1 as var(func: eq(userID, "` + blocker.Hex() + `"))
		u2 as var(func: eq(userID, "` + blocked.Hex() + `"))
	}`

	// Remove all other relationships
	dels := `uid(u1) <friends> uid(u2) .
		uid(u2) <friends> uid(u1) .
		uid(u1) <requested> uid(u2) .
		uid(u2) <requested> uid(u1) .`

	sets := `uid(u1) <blocked> uid(u2) .
		uid(u1) <created_at> "` + r.now() + `" (since=blocked) .
		uid(u1) <userID> "` + blocker.Hex() + `" .
		uid(u1) <dgraph.type> "User" .
		uid(u2) <userID> "` + blocked.Hex() + `" .
		uid(u2) <dgraph.type> "User" .`

	mu := &api.Mutation{
		DelNquads: []byte(dels),
		SetNquads: []byte(sets),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// UnblockUser removes a block relationship
func (r *DgraphRepository) UnblockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	q := `query {
		u1 as var(func: eq(userID, "` + blocker.Hex() + `"))
		u2 as var(func: eq(userID, "` + blocked.Hex() + `"))
	}`

	dels := `uid(u1) <blocked> uid(u2) .`

	mu := &api.Mutation{
		DelNquads: []byte(dels),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// GetFriendIDs returns a list of friend user IDs
func (r *DgraphRepository) GetFriendIDs(ctx context.Context, userID primitive.ObjectID) ([]string, error) {
	q := `query Friends($id: string) {
		u(func: eq(userID, $id)) {
			friends {
				userID
			}
		}
	}`

	vars := map[string]string{"$id": userID.Hex()}
	resp, err := r.client.NewTxn().QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, err
	}

	type User struct {
		Friends []struct {
			UserID string `json:"userID"`
		} `json:"friends"`
	}
	type Root struct {
		U []User `json:"u"`
	}

	var root Root
	if err := json.Unmarshal(resp.Json, &root); err != nil {
		return nil, err
	}

	var ids []string
	if len(root.U) > 0 {
		for _, f := range root.U[0].Friends {
			ids = append(ids, f.UserID)
		}
	}

	return ids, nil
}

func (r *DgraphRepository) now() string {
	return time.Now().Format(time.RFC3339)
}
