package validation

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidUserID     = errors.New("invalid user ID")
	ErrInvalidFriendID   = errors.New("invalid friend ID")
	ErrCannotFriendSelf  = errors.New("cannot send friend request to yourself")
	ErrCannotBlockSelf   = errors.New("cannot block yourself")
	ErrEmptySearchQuery  = errors.New("search query cannot be empty")
	ErrInvalidPageNumber = errors.New("page number must be positive")
	ErrInvalidLimit      = errors.New("limit must be positive")
)

// Validator provides validation for friendship operations
type Validator struct{}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateFriendRequest validates a friend request
func (v *Validator) ValidateFriendRequest(requesterID, receiverID primitive.ObjectID) error {
	if requesterID.IsZero() {
		return ErrInvalidUserID
	}
	if receiverID.IsZero() {
		return ErrInvalidFriendID
	}
	if requesterID == receiverID {
		return ErrCannotFriendSelf
	}
	return nil
}

// ValidateBlock validates a block request
func (v *Validator) ValidateBlock(blockerID, blockedID primitive.ObjectID) error {
	if blockerID.IsZero() {
		return ErrInvalidUserID
	}
	if blockedID.IsZero() {
		return ErrInvalidFriendID
	}
	if blockerID == blockedID {
		return ErrCannotBlockSelf
	}
	return nil
}

// ValidateSearch validates a search request
func (v *Validator) ValidateSearch(userID primitive.ObjectID, query string, limit int64) error {
	if userID.IsZero() {
		return ErrInvalidUserID
	}
	if query == "" {
		return ErrEmptySearchQuery
	}
	if limit <= 0 {
		return ErrInvalidLimit
	}
	return nil
}

// ValidatePagination validates pagination parameters
func (v *Validator) ValidatePagination(page, limit int64) error {
	if page <= 0 {
		return ErrInvalidPageNumber
	}
	if limit <= 0 {
		return ErrInvalidLimit
	}
	return nil
}

// ValidateUserIDs validates two user IDs for a relationship operation
func (v *Validator) ValidateUserIDs(userID1, userID2 primitive.ObjectID) error {
	if userID1.IsZero() {
		return ErrInvalidUserID
	}
	if userID2.IsZero() {
		return ErrInvalidFriendID
	}
	return nil
}
