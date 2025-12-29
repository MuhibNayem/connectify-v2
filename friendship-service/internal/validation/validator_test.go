package validation_test

import (
	"strings"
	"testing"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/validation"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestValidator_ValidateFriendRequest(t *testing.T) {
	t.Parallel()

	v := validation.NewValidator()

	tests := []struct {
		name        string
		requesterID primitive.ObjectID
		receiverID  primitive.ObjectID
		wantErr     bool
		errContains string
	}{
		{
			name:        "Valid IDs",
			requesterID: primitive.NewObjectID(),
			receiverID:  primitive.NewObjectID(),
			wantErr:     false,
		},
		{
			name:        "Zero requester ID",
			requesterID: primitive.ObjectID{},
			receiverID:  primitive.NewObjectID(),
			wantErr:     true,
			errContains: "user",
		},
		{
			name:        "Zero receiver ID",
			requesterID: primitive.NewObjectID(),
			receiverID:  primitive.ObjectID{},
			wantErr:     true,
			errContains: "friend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateFriendRequest(tt.requesterID, tt.receiverID)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidator_ValidateFriendRequest_SameUser(t *testing.T) {
	t.Parallel()

	v := validation.NewValidator()
	userID := primitive.NewObjectID()

	err := v.ValidateFriendRequest(userID, userID)

	if err == nil {
		t.Error("Expected error for same user IDs")
	}
	if !strings.Contains(err.Error(), "yourself") {
		t.Errorf("Expected error about self-friending, got: %v", err)
	}
}

func TestValidator_ValidateBlock(t *testing.T) {
	t.Parallel()

	v := validation.NewValidator()

	tests := []struct {
		name      string
		blockerID primitive.ObjectID
		blockedID primitive.ObjectID
		wantErr   bool
	}{
		{
			name:      "Valid IDs",
			blockerID: primitive.NewObjectID(),
			blockedID: primitive.NewObjectID(),
			wantErr:   false,
		},
		{
			name:      "Zero blocker ID",
			blockerID: primitive.ObjectID{},
			blockedID: primitive.NewObjectID(),
			wantErr:   true,
		},
		{
			name:      "Zero blocked ID",
			blockerID: primitive.NewObjectID(),
			blockedID: primitive.ObjectID{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateBlock(tt.blockerID, tt.blockedID)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidator_ValidateBlock_SameUser(t *testing.T) {
	t.Parallel()

	v := validation.NewValidator()
	userID := primitive.NewObjectID()

	err := v.ValidateBlock(userID, userID)

	if err == nil {
		t.Error("Expected error for blocking self")
	}
	if !strings.Contains(err.Error(), "yourself") {
		t.Errorf("Expected error about self-blocking, got: %v", err)
	}
}

func TestValidator_ValidateSearch(t *testing.T) {
	t.Parallel()

	v := validation.NewValidator()

	tests := []struct {
		name    string
		userID  primitive.ObjectID
		query   string
		limit   int64
		wantErr bool
	}{
		{"Valid search", primitive.NewObjectID(), "john", 10, false},
		{"Empty query", primitive.NewObjectID(), "", 10, true},
		{"Zero limit", primitive.NewObjectID(), "john", 0, true},
		{"Negative limit", primitive.NewObjectID(), "john", -5, true},
		{"Zero user ID", primitive.ObjectID{}, "john", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateSearch(tt.userID, tt.query, tt.limit)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidator_ValidatePagination(t *testing.T) {
	t.Parallel()

	v := validation.NewValidator()

	tests := []struct {
		name    string
		page    int64
		limit   int64
		wantErr bool
	}{
		{"Valid pagination", 1, 10, false},
		{"Zero page", 0, 10, true},
		{"Negative page", -1, 10, true},
		{"Zero limit", 1, 0, true},
		{"Negative limit", 1, -5, true},
		{"Large limit", 1, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePagination(tt.page, tt.limit)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestValidator_ValidateUserIDs(t *testing.T) {
	t.Parallel()

	v := validation.NewValidator()

	tests := []struct {
		name    string
		userID1 primitive.ObjectID
		userID2 primitive.ObjectID
		wantErr bool
	}{
		{"Both valid", primitive.NewObjectID(), primitive.NewObjectID(), false},
		{"First zero", primitive.ObjectID{}, primitive.NewObjectID(), true},
		{"Second zero", primitive.NewObjectID(), primitive.ObjectID{}, true},
		{"Both zero", primitive.ObjectID{}, primitive.ObjectID{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateUserIDs(tt.userID1, tt.userID2)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}
