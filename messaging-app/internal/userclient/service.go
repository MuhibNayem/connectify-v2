package userclient

import (
	"context"
	"log"

	"gitlab.com/spydotech-group/shared-entity/models"
	pb "gitlab.com/spydotech-group/shared-entity/proto/user/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetUserByID fetches a user from the user-service and maps it to the shared model
func (c *Client) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{UserId: id.Hex()})
	if err != nil {
		return nil, err
	}

	return mapProtoToModel(resp.User)
}

// GetUsersByIDs fetches multiple users (Batch)
func (c *Client) GetUsersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	reqIDs := make([]string, len(ids))
	for i, id := range ids {
		reqIDs[i] = id.Hex()
	}

	resp, err := c.client.GetUsers(ctx, &pb.GetUsersRequest{UserIds: reqIDs})
	if err != nil {
		return nil, err
	}

	var users []models.User
	for _, u := range resp.Users {
		modelUser, err := mapProtoToModel(u)
		if err == nil && modelUser != nil {
			users = append(users, *modelUser)
		} else {
			log.Printf("Failed to map user %s: %v", u.Id, err)
		}
	}
	return users, nil
}
