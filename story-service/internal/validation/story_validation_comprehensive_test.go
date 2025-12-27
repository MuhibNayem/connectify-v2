package validation

import (
	"testing"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
)

func TestValidateCreateStoryRequest_AllScenarios(t *testing.T) {
	tests := []struct {
		name           string
		mediaURL       string
		mediaType      string
		privacy        models.PrivacySettingType
		allowedViewers []string
		blockedViewers []string
		wantErr        bool
		expectedErr    error
	}{
		{
			name:      "valid story with public privacy",
			mediaURL:  "https://example.com/story.jpg",
			mediaType: "image",
			privacy:   models.PrivacySettingPublic,
			wantErr:   false,
		},
		{
			name:           "valid story with custom privacy",
			mediaURL:       "https://example.com/story.mp4",
			mediaType:      "video",
			privacy:        models.PrivacySettingCustom,
			allowedViewers: []string{"user1", "user2"},
			wantErr:        false,
		},
		{
			name:           "valid story with friends except privacy",
			mediaURL:       "https://example.com/story.gif",
			mediaType:      "image",
			privacy:        models.PrivacySettingFriendsExcept,
			blockedViewers: []string{"blocked_user"},
			wantErr:        false,
		},
		{
			name:        "empty media URL",
			mediaURL:    "",
			mediaType:   "image",
			wantErr:     true,
			expectedErr: ErrMediaURLRequired,
		},
		{
			name:        "media URL too long",
			mediaURL:    string(make([]byte, 501)),
			mediaType:   "image",
			wantErr:     true,
			expectedErr: ErrMediaURLTooLong,
		},
		{
			name:        "invalid media type",
			mediaURL:    "https://example.com/story.pdf",
			mediaType:   "document",
			wantErr:     true,
			expectedErr: ErrInvalidMediaType,
		},
		{
			name:        "invalid privacy setting",
			mediaURL:    "https://example.com/story.jpg",
			mediaType:   "image",
			privacy:     "invalid_privacy",
			wantErr:     true,
			expectedErr: ErrInvalidPrivacy,
		},
		{
			name:           "too many allowed viewers",
			mediaURL:       "https://example.com/story.jpg",
			mediaType:      "image",
			allowedViewers: make([]string, 101), // 101 viewers
			wantErr:        true,
			expectedErr:    ErrTooManyViewers,
		},
		{
			name:           "too many blocked viewers",
			mediaURL:       "https://example.com/story.jpg",
			mediaType:      "image",
			blockedViewers: make([]string, 101), // 101 viewers
			wantErr:        true,
			expectedErr:    ErrTooManyViewers,
		},
		{
			name:        "whitespace only media URL",
			mediaURL:    "   ",
			mediaType:   "image",
			wantErr:     true,
			expectedErr: ErrMediaURLRequired,
		},
		{
			name:      "case insensitive media type validation",
			mediaURL:  "https://example.com/story.jpg",
			mediaType: "IMAGE", // uppercase
			wantErr:   false,
		},
		{
			name:      "video media type validation",
			mediaURL:  "https://example.com/story.mp4",
			mediaType: "VIDEO", // uppercase
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateStoryRequest(
				tt.mediaURL,
				tt.mediaType,
				tt.privacy,
				tt.allowedViewers,
				tt.blockedViewers,
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateReactionType_AllScenarios(t *testing.T) {
	tests := []struct {
		name         string
		reactionType string
		wantErr      bool
	}{
		{"valid like", "like", false},
		{"valid love", "love", false},
		{"valid laugh", "laugh", false},
		{"valid wow", "wow", false},
		{"valid sad", "sad", false},
		{"valid angry", "angry", false},
		{"valid fire", "fire", false},
		{"valid heart", "heart", false},
		{"case insensitive like", "LIKE", false},
		{"case insensitive fire", "FiRe", false},
		{"invalid reaction", "invalid", true},
		{"empty reaction", "", true},
		{"numeric reaction", "123", true},
		{"special characters", "!@#", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReactionType(tt.reactionType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidReactionType, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeString_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal string", "hello", "hello"},
		{"string with spaces", "  hello world  ", "hello world"},
		{"empty string", "", ""},
		{"only spaces", "    ", ""},
		{"tabs and newlines", "\t\nhello\n\t", "hello"},
		{"unicode characters", "  héllo wørld  ", "héllo wørld"},
		{"special characters", "  hello@world!  ", "hello@world!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkValidateCreateStoryRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateCreateStoryRequest(
			"https://example.com/story.jpg",
			"image",
			models.PrivacySettingPublic,
			[]string{"user1", "user2"},
			[]string{},
		)
	}
}

func BenchmarkValidateReactionType(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateReactionType("like")
	}
}
