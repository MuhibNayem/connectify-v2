package validation

import (
	"errors"
	"net/url"
	"strings"
)

// Profile validation functions

var (
	ErrBioTooLong      = errors.New("bio must be 500 characters or less")
	ErrInvalidWebsite  = errors.New("invalid website URL format")
	ErrLocationTooLong = errors.New("location must be 100 characters or less")
	ErrFullNameTooLong = errors.New("full name must be 100 characters or less")
)

// ValidateProfileUpdate validates profile update fields
func ValidateProfileUpdate(fullName, bio, website, location string) error {
	// Full name validation
	if len(fullName) > 100 {
		return ErrFullNameTooLong
	}
	
	// Bio validation
	if len(bio) > 500 {
		return ErrBioTooLong
	}
	
	// Website validation
	if website != "" {
		if !isValidURL(website) {
			return ErrInvalidWebsite
		}
	}
	
	// Location validation
	if len(location) > 100 {
		return ErrLocationTooLong
	}
	
	return nil
}

// ValidateAvatarURL validates avatar URL format
func ValidateAvatarURL(avatarURL string) error {
	if avatarURL == "" {
		return nil // Empty is OK
	}
	
	if !isValidURL(avatarURL) {
		return errors.New("invalid avatar URL format")
	}
	
	// Check if it's an image URL (basic check)
	lower := strings.ToLower(avatarURL)
	validExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"}
	
	hasValidExtension := false
	for _, ext := range validExtensions {
		if strings.Contains(lower, ext) {
			hasValidExtension = true
			break
		}
	}
	
	if !hasValidExtension {
		return errors.New("avatar URL must point to a valid image file")
	}
	
	return nil
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	if str == "" {
		return false
	}
	
	// Add scheme if missing
	if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		str = "https://" + str
	}
	
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}