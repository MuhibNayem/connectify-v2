package validation

import (
	"errors"
	"strings"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

var (
	ErrTitleRequired      = errors.New("title is required")
	ErrTitleTooLong       = errors.New("title must be 200 characters or less")
	ErrDescriptionTooLong = errors.New("description must be 5000 characters or less")
	ErrPriceInvalid       = errors.New("price must be greater than zero")
	ErrCurrencyRequired   = errors.New("currency is required")
	ErrImagesRequired     = errors.New("at least one image is required")
	ErrLocationRequired   = errors.New("location is required")
	ErrCategoryRequired   = errors.New("category ID is required")
	ErrInvalidTags        = errors.New("too many tags (max 10)")
)

// ValidateCreateProductRequest validates a create product request
func ValidateCreateProductRequest(req *models.CreateProductRequest) error {
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

	// Price validation
	if req.Price <= 0 {
		return ErrPriceInvalid
	}

	// Currency validation
	if strings.TrimSpace(req.Currency) == "" {
		return ErrCurrencyRequired
	}

	// Images validation
	if len(req.Images) == 0 {
		return ErrImagesRequired
	}

	// Check empty strings in images
	for _, img := range req.Images {
		if strings.TrimSpace(img) == "" {
			return errors.New("image URL cannot be empty")
		}
	}

	// Location validation
	if strings.TrimSpace(req.Location) == "" {
		return ErrLocationRequired
	}

	// Category validation
	if strings.TrimSpace(req.CategoryID) == "" {
		return ErrCategoryRequired
	}

	// Tags validation
	if len(req.Tags) > 10 {
		return ErrInvalidTags
	}

	return nil
}
