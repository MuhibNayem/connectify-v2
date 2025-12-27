package validation

import (
	"errors"
	"strings"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

var (
	ErrMediaURLRequired    = errors.New("media URL is required")
	ErrMediaURLTooLong     = errors.New("media URL must be less than 500 characters")
	ErrInvalidMediaType    = errors.New("media type must be 'image' or 'video'")
	ErrInvalidPrivacy      = errors.New("invalid privacy setting")
	ErrTooManyViewers      = errors.New("too many viewers specified (max 100)")
	ErrInvalidReactionType = errors.New("invalid reaction type")
)

var ValidMediaTypes = map[string]bool{
	"image": true,
	"video": true,
}

var ValidPrivacySettings = map[models.PrivacySettingType]bool{
	models.PrivacySettingPublic:       true,
	models.PrivacySettingFriends:      true,
	models.PrivacySettingCustom:       true,
	models.PrivacySettingFriendsExcept: true,
}

var ValidReactionTypes = map[string]bool{
	"like":    true,
	"love":    true,
	"laugh":   true,
	"wow":     true,
	"sad":     true,
	"angry":   true,
	"fire":    true,
	"heart":   true,
}

func ValidateCreateStoryRequest(mediaURL, mediaType string, privacy models.PrivacySettingType, allowedViewers, blockedViewers []string) error {
	mediaURL = strings.TrimSpace(mediaURL)
	if mediaURL == "" {
		return ErrMediaURLRequired
	}
	if len(mediaURL) > 500 {
		return ErrMediaURLTooLong
	}

	if !ValidMediaTypes[strings.ToLower(mediaType)] {
		return ErrInvalidMediaType
	}

	if privacy != "" && !ValidPrivacySettings[privacy] {
		return ErrInvalidPrivacy
	}

	if len(allowedViewers) > 100 {
		return ErrTooManyViewers
	}
	if len(blockedViewers) > 100 {
		return ErrTooManyViewers
	}

	return nil
}

func ValidateReactionType(reactionType string) error {
	if !ValidReactionTypes[strings.ToLower(reactionType)] {
		return ErrInvalidReactionType
	}
	return nil
}

func SanitizeString(input string) string {
	return strings.TrimSpace(input)
}