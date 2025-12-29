package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/service"
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
			slog.Error("Failed to setup Dgraph schema for events-service", "error", err)
		}
	}()

	return repo, nil
}

func (r *DgraphRepository) setupSchema() error {
	schema := `
		type User {
			userID
			friends
			going
			interested_in
		}
		type Event {
			eventID
			has_category
			start_date
		}
		type Category {
			name
		}

		userID: string @index(exact) @upsert .
		eventID: string @index(exact) @upsert .
		name: string @index(exact) @upsert .
		
		friends: [uid] @reverse .
		going: [uid] @reverse .
		interested_in: [uid] @reverse .
		has_category: [uid] @reverse .
		
		start_date: datetime @index(hour) .
		created_at: datetime .
		since: datetime .
	`
	return r.client.Alter(context.Background(), &api.Operation{Schema: schema})
}

// Ensure implementation
var _ GraphClient = (*DgraphRepository)(nil)

// AddAttendee adds a relationship (:User)-[:GOING]->(:Event)
func (r *DgraphRepository) AddAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error {
	q := `query {
		u as var(func: eq(userID, "` + userID.Hex() + `"))
		e as var(func: eq(eventID, "` + eventID.Hex() + `"))
	}`

	mu := &api.Mutation{
		SetNquads: []byte(`uid(u) <going> uid(e) .
			uid(u) <userID> "` + userID.Hex() + `" .
			uid(u) <dgraph.type> "User" .
			uid(e) <eventID> "` + eventID.Hex() + `" .
			uid(e) <dgraph.type> "Event" .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// RemoveAttendee removes the relationship (:User)-[:GOING]->(:Event)
func (r *DgraphRepository) RemoveAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error {
	q := `query {
		u as var(func: eq(userID, "` + userID.Hex() + `"))
		e as var(func: eq(eventID, "` + eventID.Hex() + `"))
	}`

	mu := &api.Mutation{
		DelNquads: []byte(`uid(u) <going> uid(e) .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// GetFriendsGoing returns IDs of friends who are going to the event
func (r *DgraphRepository) GetFriendsGoing(ctx context.Context, userID, eventID primitive.ObjectID) ([]string, error) {
	// Query: User -> Friends -> Going -> Event(id)
	q := `query FriendsGoing($userid: string, $eventid: string) {
		var(func: eq(userID, $userid)) {
			friends {
				F as uid
			}
		}
		var(func: eq(eventID, $eventid)) {
			E as uid
		}
		
		# Find friends who are going to event E
		data(func: uid(F)) @filter(uid_in(going, uid(E))) {
			userID
		}
	}`

	vars := map[string]string{
		"$userid":  userID.Hex(),
		"$eventid": eventID.Hex(),
	}

	resp, err := r.client.NewTxn().QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, err
	}

	type User struct {
		UserID string `json:"userID"`
	}
	type Root struct {
		Data []User `json:"data"`
	}

	var root Root
	if err := json.Unmarshal(resp.Json, &root); err != nil {
		return nil, err
	}

	var ids []string
	for _, u := range root.Data {
		ids = append(ids, u.UserID)
	}
	return ids, nil
}

// GetRecommendedEventsFromGraph returns FB-scale personalized event recommendations
func (r *DgraphRepository) GetRecommendedEventsFromGraph(ctx context.Context, userID string, limit int) ([]service.GraphRecommendation, error) {
	// Complex recommendation logic translated to DQL
	// 1. Direct friends going
	// 2. FoF going
	// 3. Category interest

	q := `query Recommend($userid: string) {
		var(func: eq(userID, $userid)) {
			friends {
				F as uid
				going {
					EF as uid # Events friends are going to
				}
				friends {
					FOF as uid
					going {
						EFOF as uid # Events FoF are going to
					}
				}
			}
			interested_in {
				~has_category {
					ECAT as uid # Events matching category
				}
			}
		}

		# Filter events (e.g. future only) - requires fetching start_date
		# For now assuming simple existence
		
		# Calculate scores for candidate events (union of EF, EFOF, ECAT)
		var(func: uid(EF, EFOF, ECAT)) {
			# Score components
			
			# Count friends going
			c_friends as count(~going @filter(uid(F)))
			
			# Count FoF going
			c_fof as count(~going @filter(uid(FOF)))
			
			# Category match (boolean-ish, check if uid is in ECAT)
			# math condition
			
			score as math(c_friends * 10 + c_fof * 3) # Simplified, assume category match adds +5 if needed logic
		}
		
		# Return top events
		data(func: uid(EF, EFOF, ECAT), orderdesc: val(score), first: ` + fmt.Sprintf("%d", limit) + `) {
			eventID
			val(score)
			# Return details for client to construct response
			friends_going: ~going @filter(uid(F)) {
				userID
			}
			fof_going: ~going @filter(uid(FOF)) {
				userID
			}
			# Check category match
			has_category {
				~interested_in @filter(uid_in(userID, $userid)) {
					name # Just to have non-empty result if matched
				}
			}
		}
	}`

	vars := map[string]string{"$userid": userID}
	resp, err := r.client.NewTxn().QueryWithVars(ctx, q, vars)
	if err != nil {
		return nil, err
	}

	type IDStruct struct {
		UserID string `json:"userID"`
	}
	type EventData struct {
		EventID      string     `json:"eventID"`
		Score        float64    `json:"val(score)"`
		FriendsGoing []IDStruct `json:"friends_going"`
		FoFGoing     []IDStruct `json:"fof_going"`
		HasCategory  []struct {
			ReverseInterested []struct {
				Name string `json:"name"`
			} `json:"~interested_in"`
		} `json:"has_category"`
	}
	type Root struct {
		Data []EventData `json:"data"`
	}

	var root Root
	if err := json.Unmarshal(resp.Json, &root); err != nil {
		return nil, err
	}

	var recommendations []service.GraphRecommendation
	for _, e := range root.Data {
		rec := service.GraphRecommendation{
			EventID: e.EventID,
			Score:   e.Score,
		}
		for _, f := range e.FriendsGoing {
			rec.FriendsGoing = append(rec.FriendsGoing, f.UserID)
		}
		for _, f := range e.FoFGoing {
			rec.FoFGoing = append(rec.FoFGoing, f.UserID)
		}

		// Category match check
		matched := false
		for _, hc := range e.HasCategory {
			if len(hc.ReverseInterested) > 0 {
				matched = true
				break
			}
		}
		if matched {
			rec.CategoryMatch = true
			rec.Score += 5.0 // Manual adjustment if not in math
		}
		rec.TotalConnections = len(rec.FriendsGoing) + len(rec.FoFGoing)

		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// AddUserInterest tracks user interest in event categories
func (r *DgraphRepository) AddUserInterest(ctx context.Context, userID, category string) error {
	q := `query {
		u as var(func: eq(userID, "` + userID + `"))
		c as var(func: eq(name, "` + category + `"))
	}`

	mu := &api.Mutation{
		SetNquads: []byte(`uid(u) <interested_in> uid(c) .
			uid(u) <userID> "` + userID + `" .
			uid(u) <dgraph.type> "User" .
			uid(c) <name> "` + category + `" .
			uid(c) <dgraph.type> "Category" .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// SetEventCategory links an event to a category for interest matching
func (r *DgraphRepository) SetEventCategory(ctx context.Context, eventID, category string) error {
	q := `query {
		e as var(func: eq(eventID, "` + eventID + `"))
		c as var(func: eq(name, "` + category + `"))
	}`

	mu := &api.Mutation{
		SetNquads: []byte(`uid(e) <has_category> uid(c) .
			uid(e) <eventID> "` + eventID + `" .
			uid(e) <dgraph.type> "Event" .
			uid(c) <name> "` + category + `" .
			uid(c) <dgraph.type> "Category" .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// AddFriendship creates bidirectional FRIEND relationship
func (r *DgraphRepository) AddFriendship(ctx context.Context, userID1, userID2 string) error {
	q := `query {
		u1 as var(func: eq(userID, "` + userID1 + `"))
		u2 as var(func: eq(userID, "` + userID2 + `"))
	}`

	mu := &api.Mutation{
		SetNquads: []byte(`uid(u1) <friends> uid(u2) .
			uid(u1) <userID> "` + userID1 + `" .
			uid(u2) <userID> "` + userID2 + `" .
			uid(u2) <friends> uid(u1) .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// RemoveFriendship removes FRIEND relationship
func (r *DgraphRepository) RemoveFriendship(ctx context.Context, userID1, userID2 string) error {
	q := `query {
		u1 as var(func: eq(userID, "` + userID1 + `"))
		u2 as var(func: eq(userID, "` + userID2 + `"))
	}`

	mu := &api.Mutation{
		DelNquads: []byte(`uid(u1) <friends> uid(u2) .
			uid(u2) <friends> uid(u1) .`),
	}

	req := &api.Request{
		Query:     q,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := r.client.NewTxn().Do(ctx, req)
	return err
}

// GetMutualFriendsCount returns count of mutual friends between user and event host
func (r *DgraphRepository) GetMutualFriendsCount(ctx context.Context, userID, hostID string) (int, error) {
	// Simplified Mutual Query
	q := `query Mutual($userid: string, $hostid: string) {
		var(func: eq(userID, $userid)) {
			friends {
				F as uid
			}
		}
		
		me(func: eq(userID, $hostid)) {
			count(friends @filter(uid(F)))
		}
	}`

	vars := map[string]string{
		"$userid": userID,
		"$hostid": hostID,
	}

	resp, err := r.client.NewTxn().QueryWithVars(ctx, q, vars)
	if err != nil {
		return 0, err
	}

	type Root struct {
		Me []struct {
			Count int `json:"count(friends)"`
		} `json:"me"`
	}

	var root Root
	if err := json.Unmarshal(resp.Json, &root); err != nil {
		return 0, err
	}

	if len(root.Me) > 0 {
		return root.Me[0].Count, nil
	}
	return 0, nil
}
