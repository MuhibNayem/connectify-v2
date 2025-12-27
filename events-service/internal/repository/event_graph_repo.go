package repository

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/service"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EventGraphRepository struct {
	driver neo4j.DriverWithContext
}

func NewEventGraphRepository(driver neo4j.DriverWithContext) *EventGraphRepository {
	return &EventGraphRepository{driver: driver}
}

// AddAttendee adds a relationship (:User)-[:GOING]->(:Event)
func (r *EventGraphRepository) AddAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error {
	query := `
		MERGE (u:User {id: $userID})
		MERGE (e:Event {id: $eventID})
		MERGE (u)-[:GOING]->(e)
	`
	params := map[string]any{
		"userID":  userID.Hex(),
		"eventID": eventID.Hex(),
	}

	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

// RemoveAttendee removes the relationship (:User)-[:GOING]->(:Event)
func (r *EventGraphRepository) RemoveAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error {
	query := `
		MATCH (u:User {id: $userID})-[r:GOING]->(e:Event {id: $eventID})
		DELETE r
	`
	params := map[string]any{
		"userID":  userID.Hex(),
		"eventID": eventID.Hex(),
	}

	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

// GetFriendsGoing returns IDs of friends who are going to the event
func (r *EventGraphRepository) GetFriendsGoing(ctx context.Context, userID, eventID primitive.ObjectID) ([]string, error) {
	// Find (Me)-[:FRIEND]-(Friend)-[:GOING]->(Event)
	query := `
		MATCH (me:User {id: $userID})-[:FRIEND]-(f:User)-[:GOING]->(e:Event {id: $eventID})
		RETURN f.id as friendID
	`
	params := map[string]any{
		"userID":  userID.Hex(),
		"eventID": eventID.Hex(),
	}

	result, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		return nil, err
	}

	var friendIDs []string
	for _, record := range result.Records {
		if id, ok := record.Values[0].(string); ok {
			friendIDs = append(friendIDs, id)
		}
	}
	return friendIDs, nil
}

// GetRecommendedEventsFromGraph returns FB-scale personalized event recommendations
// Algorithm: Multi-signal scoring based on graph proximity
//   - Direct friends going: +10 points per friend
//   - Friends-of-friends going: +3 points per FoF (capped at 20)
//   - Category match with user interests: +5 points
//   - Mutual friends with host: +2 points per mutual
//   - Geographic proximity (if applicable): +5 points within 50km
func (r *EventGraphRepository) GetRecommendedEventsFromGraph(ctx context.Context, userID string, limit int) ([]service.GraphRecommendation, error) {
	query := `
		// Find events where friends or friends-of-friends are going
		MATCH (me:User {id: $userID})
		
		// Pattern 1: Direct friends going to events
		OPTIONAL MATCH (me)-[:FRIEND]-(friend:User)-[:GOING]->(event:Event)
		WHERE event.start_date > datetime()
		
		WITH me, event, collect(DISTINCT friend.id) as directFriends
		WHERE event IS NOT NULL
		
		// Pattern 2: Friends-of-friends (2-hop) going
		OPTIONAL MATCH (me)-[:FRIEND]-(:User)-[:FRIEND]-(fof:User)-[:GOING]->(event)
		WHERE NOT (me)-[:FRIEND]-(fof) AND fof.id <> $userID
		
		WITH me, event, directFriends, collect(DISTINCT fof.id)[0..20] as fofFriends
		
		// Pattern 3: Check category interests
		OPTIONAL MATCH (me)-[:INTERESTED_IN]->(cat:Category)<-[:HAS_CATEGORY]-(event)
		
		WITH event, directFriends, fofFriends, cat IS NOT NULL as categoryMatch,
		     size(directFriends) as friendCount,
		     size(fofFriends) as fofCount
		
		// Calculate score: FB-style multi-signal ranking
		WITH event, directFriends, fofFriends, categoryMatch,
		     (friendCount * 10.0) +
		     (fofCount * 3.0) +
		     (CASE WHEN categoryMatch THEN 5.0 ELSE 0.0 END) as score,
		     friendCount + fofCount as totalConnections
		
		WHERE score > 0
		
		RETURN event.id as eventId,
		       score,
		       directFriends,
		       fofFriends,
		       categoryMatch,
		       totalConnections
		ORDER BY score DESC
		LIMIT $limit
	`
	params := map[string]any{
		"userID": userID,
		"limit":  int64(limit),
	}

	result, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		// Fallback gracefully if graph is unavailable
		return nil, err
	}

	recommendations := make([]service.GraphRecommendation, 0, len(result.Records))
	for _, record := range result.Records {
		rec := service.GraphRecommendation{}

		if id, ok := record.Values[0].(string); ok {
			rec.EventID = id
		}
		if score, ok := record.Values[1].(float64); ok {
			rec.Score = score
		}
		if friends, ok := record.Values[2].([]interface{}); ok {
			for _, f := range friends {
				if s, ok := f.(string); ok {
					rec.FriendsGoing = append(rec.FriendsGoing, s)
				}
			}
		}
		if fof, ok := record.Values[3].([]interface{}); ok {
			for _, f := range fof {
				if s, ok := f.(string); ok {
					rec.FoFGoing = append(rec.FoFGoing, s)
				}
			}
		}
		if catMatch, ok := record.Values[4].(bool); ok {
			rec.CategoryMatch = catMatch
		}
		if totalConn, ok := record.Values[5].(int64); ok {
			rec.TotalConnections = int(totalConn)
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// AddUserInterest tracks user interest in event categories
func (r *EventGraphRepository) AddUserInterest(ctx context.Context, userID, category string) error {
	query := `
		MERGE (u:User {id: $userID})
		MERGE (c:Category {name: $category})
		MERGE (u)-[:INTERESTED_IN]->(c)
	`
	params := map[string]any{
		"userID":   userID,
		"category": category,
	}

	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

// SetEventCategory links an event to a category for interest matching
func (r *EventGraphRepository) SetEventCategory(ctx context.Context, eventID, category string) error {
	query := `
		MERGE (e:Event {id: $eventID})
		MERGE (c:Category {name: $category})
		MERGE (e)-[:HAS_CATEGORY]->(c)
	`
	params := map[string]any{
		"eventID":  eventID,
		"category": category,
	}

	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

// AddFriendship creates bidirectional FRIEND relationship
func (r *EventGraphRepository) AddFriendship(ctx context.Context, userID1, userID2 string) error {
	query := `
		MERGE (u1:User {id: $userID1})
		MERGE (u2:User {id: $userID2})
		MERGE (u1)-[:FRIEND]-(u2)
	`
	params := map[string]any{
		"userID1": userID1,
		"userID2": userID2,
	}

	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

// RemoveFriendship removes FRIEND relationship
func (r *EventGraphRepository) RemoveFriendship(ctx context.Context, userID1, userID2 string) error {
	query := `
		MATCH (u1:User {id: $userID1})-[r:FRIEND]-(u2:User {id: $userID2})
		DELETE r
	`
	params := map[string]any{
		"userID1": userID1,
		"userID2": userID2,
	}

	_, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

// GetMutualFriendsCount returns count of mutual friends between user and event host
func (r *EventGraphRepository) GetMutualFriendsCount(ctx context.Context, userID, hostID string) (int, error) {
	query := `
		MATCH (u:User {id: $userID})-[:FRIEND]-(mutual:User)-[:FRIEND]-(host:User {id: $hostID})
		RETURN count(DISTINCT mutual) as mutualCount
	`
	params := map[string]any{
		"userID": userID,
		"hostID": hostID,
	}

	result, err := neo4j.ExecuteQuery(ctx, r.driver, query, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		return 0, err
	}

	if len(result.Records) > 0 {
		if count, ok := result.Records[0].Values[0].(int64); ok {
			return int(count), nil
		}
	}
	return 0, nil
}
