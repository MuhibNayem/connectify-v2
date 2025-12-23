package userclient

import (
	"time"

	"gitlab.com/spydotech-group/shared-entity/models"
	pb "gitlab.com/spydotech-group/shared-entity/proto/user/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func mapProtoToModel(u *pb.User) (*models.User, error) {
	if u == nil {
		return nil, nil
	}

	oid, err := primitive.ObjectIDFromHex(u.Id)
	if err != nil {
		return nil, err
	}

	// Map basic fields
	// Note: We only map fields available in the proto response
	// Map Privacy Settings
	var privacySettings models.UserPrivacySettings
	if u.PrivacySettings != nil {
		privacySettings = models.UserPrivacySettings{
			// UserID is not in PrivacySettings proto, it's implicit from the user.ID
			DefaultPostPrivacy:      models.PrivacySettingType(u.PrivacySettings.DefaultPostPrivacy),
			CanSeeMyFriendsList:     models.PrivacySettingType(u.PrivacySettings.CanSeeMyFriendsList),
			CanSendMeFriendRequests: models.PrivacySettingType(u.PrivacySettings.CanSendMeFriendRequests),
			CanTagMeInPosts:         models.PrivacySettingType(u.PrivacySettings.CanTagMeInPosts),
		}
		if u.PrivacySettings.LastUpdated != nil {
			privacySettings.LastUpdated = u.PrivacySettings.LastUpdated.AsTime()
		}
	}

	// Map Notification Settings
	var notificationSettings models.NotificationSettings
	if u.NotificationSettings != nil {
		notificationSettings = models.NotificationSettings{
			EmailNotifications:    u.NotificationSettings.EmailNotifications,
			PushNotifications:     u.NotificationSettings.PushNotifications,
			NotifyOnFriendRequest: u.NotificationSettings.NotifyOnFriendRequest,
			NotifyOnComment:       u.NotificationSettings.NotifyOnComment,
			NotifyOnLike:          u.NotificationSettings.NotifyOnLike,
			NotifyOnTag:           u.NotificationSettings.NotifyOnTag,
			NotifyOnMessage:       u.NotificationSettings.NotifyOnMessage,
			NotifyOnBirthday:      u.NotificationSettings.NotifyOnBirthday,
			NotifyOnEventInvite:   u.NotificationSettings.NotifyOnEventInvite,
		}
	}

	var dob *time.Time
	if u.DateOfBirth != nil {
		t := u.DateOfBirth.AsTime()
		dob = &t
	}

	userModel := &models.User{
		ID:                   oid,
		Username:             u.Username,
		Email:                u.Email,
		FullName:             u.FullName,
		Avatar:               u.Avatar,
		CoverPicture:         u.CoverPicture,
		Bio:                  u.Bio,
		Gender:               u.Gender, // Ensure Gender is mapped!
		Location:             u.Location,
		PhoneNumber:          u.PhoneNumber,
		TwoFactorEnabled:     u.TwoFactorEnabled,
		EmailVerified:        u.EmailVerified,
		IsActive:             u.IsActive,
		IsEncryptionEnabled:  u.IsEncryptionEnabled,
		PrivacySettings:      privacySettings,
		NotificationSettings: notificationSettings,
		DateOfBirth:          dob,
	}

	if u.CreatedAt != nil {
		userModel.CreatedAt = u.CreatedAt.AsTime()
	}
	if u.UpdatedAt != nil {
		userModel.UpdatedAt = u.UpdatedAt.AsTime()
	}

	return userModel, nil
}
