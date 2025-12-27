package validation

import (
	"testing"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
)

func TestValidateCreateStoryRequest(t *testing.T) {
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
			name:      "valid request",
			mediaURL:  "https://example.com/story.jpg",
			mediaType: "image",
			privacy:   models.PrivacySettingPublic,
			wantErr:   false,
		},
		{
			name:        "empty media URL",
			mediaURL:    "",
			mediaType:   "image",
			wantErr:     true,
			expectedErr: ErrMediaURLRequired,
		},
		{
			name:      "media URL too long",
			mediaURL:  string(make([]byte, 501)),
			mediaType: "image",
			wantErr:   true,
			expectedErr: ErrMediaURLTooLong,
		},
		{
			name:        "invalid media type",
			mediaURL:    "https://example.com/story.mp4",
			mediaType:   "audio",
			wantErr:     true,
			expectedErr: ErrInvalidMediaType,
		},
		{
			name:      "invalid privacy setting",
			mediaURL:  "https://example.com/story.jpg",
			mediaType: "image",
			privacy:   "invalid",
			wantErr:   true,
			expectedErr: ErrInvalidPrivacy,
		},
		{
			name:           "too many allowed viewers",
			mediaURL:       "https://example.com/story.jpg",
			mediaType:      "image",
			allowedViewers: make([]string, 101),
			wantErr:        true,
			expectedErr:    ErrTooManyViewers,
		},
		{
			name:           "too many blocked viewers",
			mediaURL:       "https://example.com/story.jpg",
			mediaType:      "image",
			blockedViewers: make([]string, 101),
			wantErr:        true,
			expectedErr:    ErrTooManyViewers,
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

func TestValidateReactionType(t *testing.T) {
	tests := []struct {
		name         string
		reactionType string
		wantErr      bool
	}{
		{"valid like", "like", false},
		{"valid love", "love", false},
		{"valid fire", "fire", false},
		{"invalid reaction", "invalid", true},
		{"empty reaction", "", true},
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

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"world", "world"},
		{"", ""},
		{"  ", ""},
	}

	for _, tt := range tests {
		result := SanitizeString(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}