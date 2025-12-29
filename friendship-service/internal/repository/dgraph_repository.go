package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DgraphRepository provides graph operations using Dgraph
type DgraphRepository struct {
	client *dgo.Dgraph
	conn   *grpc.ClientConn
}

// User represents a user node in Dgraph
type User struct {
	UID       string   `json:"uid,omitempty"`
	DType     []string `json:"dgraph.type,omitempty"`
	ID        string   `json:"User.id,omitempty"`
	Friends   []User   `json:"User.friends,omitempty"`
	Requested []User   `json:"User.requested,omitempty"`
	Blocked   []User   `json:"User.blocked,omitempty"`
}

// NewDgraphRepository creates a new Dgraph repository
func NewDgraphRepository(addr string) (*DgraphRepository, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to dgraph: %w", err)
	}

	client := dgo.NewDgraphClient(api.NewDgraphClient(conn))

	repo := &DgraphRepository{
		client: client,
		conn:   conn,
	}

	// Setup schema
	if err := repo.setupSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to setup schema: %w", err)
	}

	return repo, nil
}

// setupSchema creates the Dgraph schema for friendships
func (r *DgraphRepository) setupSchema() error {
	schema := `
		User.id: string @index(exact) .
		User.friends: [uid] @reverse .
		User.requested: [uid] @reverse .
		User.blocked: [uid] .
		User.requested_at: datetime .
		User.friends_since: datetime .
		User.blocked_at: datetime .

		type User {
			User.id
			User.friends
			User.requested
			User.blocked
		}
	`

	return r.client.Alter(context.Background(), &api.Operation{Schema: schema})
}

// Close closes the Dgraph connection
func (r *DgraphRepository) Close() error {
	return r.conn.Close()
}

// getOrCreateUser finds or creates a user node by MongoDB ObjectID
func (r *DgraphRepository) getOrCreateUser(ctx context.Context, txn *dgo.Txn, userID string) (string, error) {
	// Query for existing user
	query := fmt.Sprintf(`{
		user(func: eq(User.id, "%s")) {
			uid
		}
	}`, userID)

	resp, err := txn.Query(ctx, query)
	if err != nil {
		return "", err
	}

	var result struct {
		User []struct {
			UID string `json:"uid"`
		} `json:"user"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return "", err
	}

	if len(result.User) > 0 {
		return result.User[0].UID, nil
	}

	// Create new user
	user := User{
		DType: []string{"User"},
		ID:    userID,
	}

	data, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	mu := &api.Mutation{
		SetJson:   data,
		CommitNow: false,
	}

	assigned, err := txn.Mutate(ctx, mu)
	if err != nil {
		return "", err
	}

	return assigned.Uids["blank-0"], nil
}

// SyncUser ensures a user exists in the graph
func (r *DgraphRepository) SyncUser(ctx context.Context, userID primitive.ObjectID) error {
	txn := r.client.NewTxn()
	defer txn.Discard(ctx)

	_, err := r.getOrCreateUser(ctx, txn, userID.Hex())
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

// SendRequest creates a friend request from one user to another
func (r *DgraphRepository) SendRequest(ctx context.Context, from, to primitive.ObjectID) error {
	txn := r.client.NewTxn()
	defer txn.Discard(ctx)

	fromUID, err := r.getOrCreateUser(ctx, txn, from.Hex())
	if err != nil {
		return err
	}

	toUID, err := r.getOrCreateUser(ctx, txn, to.Hex())
	if err != nil {
		return err
	}

	// Create requested edge
	nquads := fmt.Sprintf(`<%s> <User.requested> <%s> .`, fromUID, toUID)

	mu := &api.Mutation{
		SetNquads: []byte(nquads),
		CommitNow: false,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

// AcceptRequest accepts a friend request and creates friendship
func (r *DgraphRepository) AcceptRequest(ctx context.Context, from, to primitive.ObjectID) error {
	txn := r.client.NewTxn()
	defer txn.Discard(ctx)

	// Find users
	query := fmt.Sprintf(`{
		from(func: eq(User.id, "%s")) { uid }
		to(func: eq(User.id, "%s")) { uid }
	}`, from.Hex(), to.Hex())

	resp, err := txn.Query(ctx, query)
	if err != nil {
		return err
	}

	var result struct {
		From []struct {
			UID string `json:"uid"`
		} `json:"from"`
		To []struct {
			UID string `json:"uid"`
		} `json:"to"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return err
	}

	if len(result.From) == 0 || len(result.To) == 0 {
		return nil // Users don't exist
	}

	fromUID := result.From[0].UID
	toUID := result.To[0].UID

	// Delete requested edge, create friends edge (bidirectional)
	deleteNquads := fmt.Sprintf(`<%s> <User.requested> <%s> .
<%s> <User.requested> <%s> .`, fromUID, toUID, toUID, fromUID)

	setNquads := fmt.Sprintf(`<%s> <User.friends> <%s> .
<%s> <User.friends> <%s> .`, fromUID, toUID, toUID, fromUID)

	mu := &api.Mutation{
		DelNquads: []byte(deleteNquads),
		SetNquads: []byte(setNquads),
		CommitNow: false,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

// RejectRequest removes a friend request
func (r *DgraphRepository) RejectRequest(ctx context.Context, from, to primitive.ObjectID) error {
	txn := r.client.NewTxn()
	defer txn.Discard(ctx)

	query := fmt.Sprintf(`{
		from(func: eq(User.id, "%s")) { uid }
		to(func: eq(User.id, "%s")) { uid }
	}`, from.Hex(), to.Hex())

	resp, err := txn.Query(ctx, query)
	if err != nil {
		return err
	}

	var result struct {
		From []struct {
			UID string `json:"uid"`
		} `json:"from"`
		To []struct {
			UID string `json:"uid"`
		} `json:"to"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return err
	}

	if len(result.From) == 0 || len(result.To) == 0 {
		return nil
	}

	// Delete requested edge in both directions
	deleteNquads := fmt.Sprintf(`<%s> <User.requested> <%s> .
<%s> <User.requested> <%s> .`, result.From[0].UID, result.To[0].UID, result.To[0].UID, result.From[0].UID)

	mu := &api.Mutation{
		DelNquads: []byte(deleteNquads),
		CommitNow: false,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

// Unfriend removes a friendship between two users
func (r *DgraphRepository) Unfriend(ctx context.Context, user1, user2 primitive.ObjectID) error {
	txn := r.client.NewTxn()
	defer txn.Discard(ctx)

	query := fmt.Sprintf(`{
		u1(func: eq(User.id, "%s")) { uid }
		u2(func: eq(User.id, "%s")) { uid }
	}`, user1.Hex(), user2.Hex())

	resp, err := txn.Query(ctx, query)
	if err != nil {
		return err
	}

	var result struct {
		U1 []struct {
			UID string `json:"uid"`
		} `json:"u1"`
		U2 []struct {
			UID string `json:"uid"`
		} `json:"u2"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return err
	}

	if len(result.U1) == 0 || len(result.U2) == 0 {
		return nil
	}

	// Delete friends edge in both directions
	deleteNquads := fmt.Sprintf(`<%s> <User.friends> <%s> .
<%s> <User.friends> <%s> .`, result.U1[0].UID, result.U2[0].UID, result.U2[0].UID, result.U1[0].UID)

	mu := &api.Mutation{
		DelNquads: []byte(deleteNquads),
		CommitNow: false,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

// BlockUser blocks a user and removes any friendship/requests
func (r *DgraphRepository) BlockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	txn := r.client.NewTxn()
	defer txn.Discard(ctx)

	blockerUID, err := r.getOrCreateUser(ctx, txn, blocker.Hex())
	if err != nil {
		return err
	}

	blockedUID, err := r.getOrCreateUser(ctx, txn, blocked.Hex())
	if err != nil {
		return err
	}

	// Delete friends and requested edges, add blocked edge
	deleteNquads := fmt.Sprintf(`<%s> <User.friends> <%s> .
<%s> <User.friends> <%s> .
<%s> <User.requested> <%s> .
<%s> <User.requested> <%s> .`,
		blockerUID, blockedUID,
		blockedUID, blockerUID,
		blockerUID, blockedUID,
		blockedUID, blockerUID)

	setNquads := fmt.Sprintf(`<%s> <User.blocked> <%s> .`, blockerUID, blockedUID)

	mu := &api.Mutation{
		DelNquads: []byte(deleteNquads),
		SetNquads: []byte(setNquads),
		CommitNow: false,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

// UnblockUser removes a block relationship
func (r *DgraphRepository) UnblockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	txn := r.client.NewTxn()
	defer txn.Discard(ctx)

	query := fmt.Sprintf(`{
		blocker(func: eq(User.id, "%s")) { uid }
		blocked(func: eq(User.id, "%s")) { uid }
	}`, blocker.Hex(), blocked.Hex())

	resp, err := txn.Query(ctx, query)
	if err != nil {
		return err
	}

	var result struct {
		Blocker []struct {
			UID string `json:"uid"`
		} `json:"blocker"`
		Blocked []struct {
			UID string `json:"uid"`
		} `json:"blocked"`
	}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return err
	}

	if len(result.Blocker) == 0 || len(result.Blocked) == 0 {
		return nil
	}

	deleteNquads := fmt.Sprintf(`<%s> <User.blocked> <%s> .`, result.Blocker[0].UID, result.Blocked[0].UID)

	mu := &api.Mutation{
		DelNquads: []byte(deleteNquads),
		CommitNow: false,
	}

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return err
	}

	return txn.Commit(ctx)
}

// CheckFriendshipStatus returns friendship status between two users
func (r *DgraphRepository) CheckFriendshipStatus(ctx context.Context, me, other primitive.ObjectID) (areFriends, requestSent, requestReceived, blockedByMe, blockedByOther bool, err error) {
	query := fmt.Sprintf(`{
		me(func: eq(User.id, "%s")) {
			uid
			friends: User.friends @filter(eq(User.id, "%s")) { uid }
			requested: User.requested @filter(eq(User.id, "%s")) { uid }
			blocked: User.blocked @filter(eq(User.id, "%s")) { uid }
		}
		other(func: eq(User.id, "%s")) {
			uid
			requested: User.requested @filter(eq(User.id, "%s")) { uid }
			blocked: User.blocked @filter(eq(User.id, "%s")) { uid }
		}
	}`, me.Hex(), other.Hex(), other.Hex(), other.Hex(), other.Hex(), me.Hex(), me.Hex())

	resp, err := r.client.NewReadOnlyTxn().Query(ctx, query)
	if err != nil {
		return false, false, false, false, false, err
	}

	var result struct {
		Me []struct {
			UID     string `json:"uid"`
			Friends []struct {
				UID string `json:"uid"`
			} `json:"friends"`
			Requested []struct {
				UID string `json:"uid"`
			} `json:"requested"`
			Blocked []struct {
				UID string `json:"uid"`
			} `json:"blocked"`
		} `json:"me"`
		Other []struct {
			UID       string `json:"uid"`
			Requested []struct {
				UID string `json:"uid"`
			} `json:"requested"`
			Blocked []struct {
				UID string `json:"uid"`
			} `json:"blocked"`
		} `json:"other"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return false, false, false, false, false, err
	}

	if len(result.Me) > 0 {
		areFriends = len(result.Me[0].Friends) > 0
		requestSent = len(result.Me[0].Requested) > 0
		blockedByMe = len(result.Me[0].Blocked) > 0
	}

	if len(result.Other) > 0 {
		requestReceived = len(result.Other[0].Requested) > 0
		blockedByOther = len(result.Other[0].Blocked) > 0
	}

	return areFriends, requestSent, requestReceived, blockedByMe, blockedByOther, nil
}

// AreFriends checks if two users are friends
func (r *DgraphRepository) AreFriends(ctx context.Context, user1, user2 primitive.ObjectID) (bool, error) {
	query := fmt.Sprintf(`{
		user(func: eq(User.id, "%s")) {
			friends: User.friends @filter(eq(User.id, "%s")) { uid }
		}
	}`, user1.Hex(), user2.Hex())

	resp, err := r.client.NewReadOnlyTxn().Query(ctx, query)
	if err != nil {
		return false, err
	}

	var result struct {
		User []struct {
			Friends []struct {
				UID string `json:"uid"`
			} `json:"friends"`
		} `json:"user"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return false, err
	}

	if len(result.User) > 0 && len(result.User[0].Friends) > 0 {
		return true, nil
	}

	return false, nil
}

// Block is an alias for BlockUser
func (r *DgraphRepository) Block(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	return r.BlockUser(ctx, blocker, blocked)
}

// Unblock is an alias for UnblockUser
func (r *DgraphRepository) Unblock(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	return r.UnblockUser(ctx, blocker, blocked)
}
