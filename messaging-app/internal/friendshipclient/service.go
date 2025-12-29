package friendshipclient

import (
	"context"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	friendshippb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/friendship/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const defaultTimeout = 5 * time.Second

// SendRequest sends a friend request
func (c *Client) SendRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var resp *friendshippb.SendRequestResponse
	var err error

	_, cbErr := c.cb.Execute(ctx, func() (interface{}, error) {
		resp, err = c.client.SendRequest(ctx, &friendshippb.SendRequestRequest{
			RequesterId: requesterID.Hex(),
			ReceiverId:  receiverID.Hex(),
		})
		return resp, err
	})
	if cbErr != nil {
		return nil, cbErr
	}
	if err != nil {
		return nil, err
	}

	return toModelFriendship(resp.Friendship), nil
}

// RespondToRequest accepts or rejects a friend request
func (c *Client) RespondToRequest(ctx context.Context, friendshipID, receiverID primitive.ObjectID, accept bool) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.RespondToRequest(ctx, &friendshippb.RespondToRequestRequest{
			FriendshipId: friendshipID.Hex(),
			ReceiverId:   receiverID.Hex(),
			Accept:       accept,
		})
	})
	return err
}

// ListFriendships returns paginated friendships
func (c *Client) ListFriendships(ctx context.Context, userID primitive.ObjectID, status models.FriendshipStatus, page, limit int64) ([]models.PopulatedFriendship, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var resp *friendshippb.ListFriendshipsResponse
	var err error

	_, cbErr := c.cb.Execute(ctx, func() (interface{}, error) {
		resp, err = c.client.ListFriendships(ctx, &friendshippb.ListFriendshipsRequest{
			UserId: userID.Hex(),
			Status: string(status),
			Page:   page,
			Limit:  limit,
		})
		return resp, err
	})
	if cbErr != nil {
		return nil, 0, cbErr
	}
	if err != nil {
		return nil, 0, err
	}

	return toModelPopulatedFriendships(resp.Friendships), resp.Total, nil
}

// CheckFriendship checks if two users are friends
func (c *Client) CheckFriendship(ctx context.Context, userID1, userID2 primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var resp *friendshippb.CheckFriendshipResponse
	var err error

	_, cbErr := c.cb.Execute(ctx, func() (interface{}, error) {
		resp, err = c.client.CheckFriendship(ctx, &friendshippb.CheckFriendshipRequest{
			UserId1: userID1.Hex(),
			UserId2: userID2.Hex(),
		})
		return resp, err
	})
	if cbErr != nil {
		return false, cbErr
	}
	if err != nil {
		return false, err
	}
	return resp.AreFriends, nil
}

// Unfriend removes a friendship
func (c *Client) Unfriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.Unfriend(ctx, &friendshippb.UnfriendRequest{
			UserId:   userID.Hex(),
			FriendId: friendID.Hex(),
		})
	})
	return err
}

// BlockUser blocks a user
func (c *Client) BlockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.BlockUser(ctx, &friendshippb.BlockUserRequest{
			BlockerId: blockerID.Hex(),
			BlockedId: blockedID.Hex(),
		})
	})
	return err
}

// UnblockUser unblocks a user
func (c *Client) UnblockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UnblockUser(ctx, &friendshippb.UnblockUserRequest{
			BlockerId: blockerID.Hex(),
			BlockedId: blockedID.Hex(),
		})
	})
	return err
}

// GetBlockedUsers returns users blocked by the given user
func (c *Client) GetBlockedUsers(ctx context.Context, userID primitive.ObjectID) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var resp *friendshippb.GetBlockedUsersResponse
	var err error

	_, cbErr := c.cb.Execute(ctx, func() (interface{}, error) {
		resp, err = c.client.GetBlockedUsers(ctx, &friendshippb.GetBlockedUsersRequest{
			UserId: userID.Hex(),
		})
		return resp, err
	})
	if cbErr != nil {
		return nil, cbErr
	}
	if err != nil {
		return nil, err
	}

	return toModelUsers(resp.BlockedUsers), nil
}

// GetDetailedFriendshipStatus returns comprehensive status between two users
func (c *Client) GetDetailedFriendshipStatus(ctx context.Context, viewerID, otherUserID primitive.ObjectID) (*FriendshipStatusResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var resp *friendshippb.GetDetailedFriendshipStatusResponse
	var err error

	_, cbErr := c.cb.Execute(ctx, func() (interface{}, error) {
		resp, err = c.client.GetDetailedFriendshipStatus(ctx, &friendshippb.GetDetailedFriendshipStatusRequest{
			ViewerId:    viewerID.Hex(),
			OtherUserId: otherUserID.Hex(),
		})
		return resp, err
	})
	if cbErr != nil {
		return nil, cbErr
	}
	if err != nil {
		return nil, err
	}

	return toModelFriendshipStatus(resp), nil
}

// SearchFriends searches for friends matching a query
func (c *Client) SearchFriends(ctx context.Context, userID primitive.ObjectID, query string, limit int64) ([]models.UserShortResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var resp *friendshippb.SearchFriendsResponse
	var err error

	_, cbErr := c.cb.Execute(ctx, func() (interface{}, error) {
		resp, err = c.client.SearchFriends(ctx, &friendshippb.SearchFriendsRequest{
			UserId: userID.Hex(),
			Query:  query,
			Limit:  limit,
		})
		return resp, err
	})
	if cbErr != nil {
		return nil, cbErr
	}
	if err != nil {
		return nil, err
	}

	return toModelUserShorts(resp.Results), nil
}
