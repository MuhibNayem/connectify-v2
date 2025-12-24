package userclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"messaging-app/config"

	"gitlab.com/spydotech-group/shared-entity/observability"
	pb "gitlab.com/spydotech-group/shared-entity/proto/user/v1"
	"gitlab.com/spydotech-group/shared-entity/resilience"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC connection to the User service with circuit breaker protection
type Client struct {
	conn   *grpc.ClientConn
	client pb.UserServiceClient
	cb     *resilience.CircuitBreaker
}

// New creates a new User gRPC client using the configured host/port
func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	addr := net.JoinHostPort(cfg.UserServiceHost, cfg.UserServicePort)

	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		observability.GetGRPCDialOption(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to user gRPC at %s: %w", addr, err)
	}

	// Create circuit breaker with default config
	cbConfig := resilience.DefaultConfig("user-service")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &Client{
		conn:   conn,
		client: pb.NewUserServiceClient(conn),
		cb:     cb,
	}, nil
}

// Close shuts down the underlying gRPC connection
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// ==================== READ OPERATIONS ====================

// GetUser fetches a single user by ID (circuit breaker protected)
func (c *Client) GetUser(ctx context.Context, userID string) (*pb.User, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetUser(ctx, &pb.GetUserRequest{UserId: userID})
	})
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", userID, err)
	}
	return result.(*pb.GetUserResponse).User, nil
}

// GetUsers fetches multiple users by their IDs (circuit breaker protected)
func (c *Client) GetUsers(ctx context.Context, userIDs []string) ([]*pb.User, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetUsers(ctx, &pb.GetUsersRequest{UserIds: userIDs})
	})
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}
	return result.(*pb.GetUsersResponse).Users, nil
}

// ListUsers retrieves a paginated list of users with optional search
func (c *Client) ListUsers(ctx context.Context, page, limit int64, search string) (*pb.ListUsersResponse, error) {
	resp, err := c.client.ListUsers(ctx, &pb.ListUsersRequest{
		Page:   page,
		Limit:  limit,
		Search: search,
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return resp, nil
}

// GetUserStatus fetches the online/offline status of a single user
func (c *Client) GetUserStatus(ctx context.Context, userID string) (*pb.GetUserStatusResponse, error) {
	resp, err := c.client.GetUserStatus(ctx, &pb.GetUserStatusRequest{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("get user status %s: %w", userID, err)
	}
	return resp, nil
}

// GetUsersPresence fetches presence info for multiple users
func (c *Client) GetUsersPresence(ctx context.Context, userIDs []string) (map[string]*pb.UserPresence, error) {
	resp, err := c.client.GetUsersPresence(ctx, &pb.GetUsersPresenceRequest{UserIds: userIDs})
	if err != nil {
		return nil, fmt.Errorf("get users presence: %w", err)
	}
	return resp.Presence, nil
}

// ==================== WRITE OPERATIONS ====================

// UpdateUser updates user profile fields
func (c *Client) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	resp, err := c.client.UpdateUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("update user %s: %w", req.UserId, err)
	}
	return resp.User, nil
}

// UpdateEmail changes a user's email address
func (c *Client) UpdateEmail(ctx context.Context, userID, newEmail string) (bool, error) {
	resp, err := c.client.UpdateEmail(ctx, &pb.UpdateEmailRequest{
		UserId:   userID,
		NewEmail: newEmail,
	})
	if err != nil {
		return false, fmt.Errorf("update email for %s: %w", userID, err)
	}
	return resp.Success, nil
}

// UpdatePassword changes a user's password (validates current password)
func (c *Client) UpdatePassword(ctx context.Context, userID, currentPassword, newPassword string) (bool, error) {
	resp, err := c.client.UpdatePassword(ctx, &pb.UpdatePasswordRequest{
		UserId:          userID,
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	})
	if err != nil {
		return false, fmt.Errorf("update password for %s: %w", userID, err)
	}
	return resp.Success, nil
}

// ToggleTwoFactor enables or disables two-factor authentication
func (c *Client) ToggleTwoFactor(ctx context.Context, userID string, enable bool) (bool, error) {
	resp, err := c.client.ToggleTwoFactor(ctx, &pb.ToggleTwoFactorRequest{
		UserId: userID,
		Enable: enable,
	})
	if err != nil {
		return false, fmt.Errorf("toggle 2FA for %s: %w", userID, err)
	}
	return resp.Success, nil
}

// DeactivateAccount deactivates a user account
func (c *Client) DeactivateAccount(ctx context.Context, userID string) (bool, error) {
	resp, err := c.client.DeactivateAccount(ctx, &pb.DeactivateAccountRequest{UserId: userID})
	if err != nil {
		return false, fmt.Errorf("deactivate account %s: %w", userID, err)
	}
	return resp.Success, nil
}

// UpdatePublicKey updates a user's encryption public key and backup
func (c *Client) UpdatePublicKey(ctx context.Context, userID, publicKey, encryptedPrivateKey, keyBackupIV, keyBackupSalt string) (bool, error) {
	resp, err := c.client.UpdatePublicKey(ctx, &pb.UpdatePublicKeyRequest{
		UserId:              userID,
		PublicKey:           publicKey,
		EncryptedPrivateKey: encryptedPrivateKey,
		KeyBackupIv:         keyBackupIV,
		KeyBackupSalt:       keyBackupSalt,
	})
	if err != nil {
		return false, fmt.Errorf("update public key for %s: %w", userID, err)
	}
	return resp.Success, nil
}

// UpdatePrivacySettings updates a user's privacy settings
func (c *Client) UpdatePrivacySettings(ctx context.Context, userID string, settings map[string]string) (bool, error) {
	req := &pb.UpdatePrivacySettingsRequest{UserId: userID}
	if v, ok := settings["default_post_privacy"]; ok {
		req.DefaultPostPrivacy = v
	}
	if v, ok := settings["can_see_my_friends_list"]; ok {
		req.CanSeeMyFriendsList = v
	}
	if v, ok := settings["can_send_me_friend_requests"]; ok {
		req.CanSendMeFriendRequests = v
	}
	if v, ok := settings["can_tag_me_in_posts"]; ok {
		req.CanTagMeInPosts = v
	}
	resp, err := c.client.UpdatePrivacySettings(ctx, req)
	if err != nil {
		return false, fmt.Errorf("update privacy settings for %s: %w", userID, err)
	}
	return resp.Success, nil
}

// UpdateNotificationSettings updates a user's notification preferences
func (c *Client) UpdateNotificationSettings(ctx context.Context, req *pb.UpdateNotificationSettingsRequest) (bool, error) {
	resp, err := c.client.UpdateNotificationSettings(ctx, req)
	if err != nil {
		return false, fmt.Errorf("update notification settings for %s: %w", req.UserId, err)
	}
	return resp.Success, nil
}
