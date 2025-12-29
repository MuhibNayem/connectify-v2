package repository

import (
	"context"
	"fmt"
	"time"

	userv1 "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	client userv1.UserServiceClient
	conn   *grpc.ClientConn
}

func NewUserClient(address string) (*UserClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}
	client := userv1.NewUserServiceClient(conn)
	return &UserClient{client: client, conn: conn}, nil
}

func (c *UserClient) Close() error {
	return c.conn.Close()
}

func (c *UserClient) AddFriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.AddFriend(ctx, &userv1.AddFriendRequest{
		UserId:   userID.Hex(),
		FriendId: friendID.Hex(),
	})
	return err
}

func (c *UserClient) RemoveFriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.RemoveFriend(ctx, &userv1.RemoveFriendRequest{
		UserId:   userID.Hex(),
		FriendId: friendID.Hex(),
	})
	return err
}

func (c *UserClient) FindUserByID(ctx context.Context, userID primitive.ObjectID) (*userv1.User, error) {
	// Not strictly needed for logic but good to have for block list enrichment if desired.
	// For now unimplemented or basic.
	return nil, nil
}
