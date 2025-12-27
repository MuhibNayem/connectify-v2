package validation

import (
	"errors"
	"strings"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

var (
	ErrTitleRequired      = errors.New("title is required")
	ErrTitleTooLong       = errors.New("title must be 200 characters or less")
	ErrDescriptionTooLong = errors.New("description must be 5000 characters or less")
	ErrInvalidDateRange   = errors.New("end date must be after start date")
	ErrPastStartDate      = errors.New("start date cannot be in the past")
	ErrInvalidPrivacy     = errors.New("invalid privacy setting")
	ErrInvalidCategory    = errors.New("invalid category")
)

// ValidPrivacies defines allowed privacy values
var ValidPrivacies = map[models.EventPrivacy]bool{
	models.EventPrivacyPublic:  true,
	models.EventPrivacyPrivate: true,
	models.EventPrivacyFriends: true,
}

// ValidCategories defines allowed event categories
var ValidCategories = map[string]bool{
	"music":      true,
	"sports":     true,
	"arts":       true,
	"food":       true,
	"tech":       true,
	"business":   true,
	"education":  true,
	"gaming":     true,
	"health":     true,
	"social":     true,
	"travel":     true,
	"networking": true,
	"other":      true,
}

// ValidateCreateEventRequest validates a create event request
func ValidateCreateEventRequest(req *models.CreateEventRequest) error {
	// Title validation
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrTitleRequired
	}
	if len(title) > 200 {
		return ErrTitleTooLong
	}

	// Description validation
	if len(req.Description) > 5000 {
		return ErrDescriptionTooLong
	}

	// Date validation
	if !req.StartDate.IsZero() && req.StartDate.Before(time.Now().Add(-1*time.Hour)) {
		return ErrPastStartDate
	}
	if !req.StartDate.IsZero() && !req.EndDate.IsZero() && req.EndDate.Before(req.StartDate) {
		return ErrInvalidDateRange
	}

	// Privacy validation
	if req.Privacy != "" && !ValidPrivacies[req.Privacy] {
		return ErrInvalidPrivacy
	}

	// Category validation
	if req.Category != "" {
		cat := strings.ToLower(req.Category)
		if !ValidCategories[cat] {
			return ErrInvalidCategory
		}
	}

	return nil
}

// ValidateUpdateEventRequest validates an update event request
func ValidateUpdateEventRequest(req *models.UpdateEventRequest) error {
	// Title validation (if provided)
	if req.Title != "" {
		title := strings.TrimSpace(req.Title)
		if len(title) > 200 {
			return ErrTitleTooLong
		}
	}

	// Description validation
	if len(req.Description) > 5000 {
		return ErrDescriptionTooLong
	}

	// Date validation
	if req.StartDate != nil && req.EndDate != nil && req.EndDate.Before(*req.StartDate) {
		return ErrInvalidDateRange
	}

	// Privacy validation
	if req.Privacy != "" && !ValidPrivacies[req.Privacy] {
		return ErrInvalidPrivacy
	}

	// Category validation
	if req.Category != "" {
		cat := strings.ToLower(req.Category)
		if !ValidCategories[cat] {
			return ErrInvalidCategory
		}
	}

	return nil
}
