package friendshipclient

import (
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	friendshippb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/friendship/v1"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============================================================================
// Proto to Model Converters
// ============================================================================

func toModelFriendship(pb *friendshippb.Friendship) *models.Friendship {
	if pb == nil {
		return nil
	}
	id, _ := primitive.ObjectIDFromHex(pb.Id)
	reqID, _ := primitive.ObjectIDFromHex(pb.RequesterId)
	recID, _ := primitive.ObjectIDFromHex(pb.ReceiverId)
	return &models.Friendship{
		ID:          id,
		RequesterID: reqID,
		ReceiverID:  recID,
		Status:      models.FriendshipStatus(pb.Status),
		CreatedAt:   fromTimestamp(pb.CreatedAt),
		UpdatedAt:   fromTimestamp(pb.UpdatedAt),
	}
}

func toModelPopulatedFriendship(pb *friendshippb.PopulatedFriendship) *models.PopulatedFriendship {
	if pb == nil {
		return nil
	}
	id, _ := primitive.ObjectIDFromHex(pb.Id)
	reqID, _ := primitive.ObjectIDFromHex(pb.RequesterId)
	recID, _ := primitive.ObjectIDFromHex(pb.ReceiverId)

	return &models.PopulatedFriendship{
		ID:            id,
		RequesterID:   reqID,
		ReceiverID:    recID,
		Status:        models.FriendshipStatus(pb.Status),
		RequesterInfo: toModelSafeUser(pb.RequesterInfo),
		ReceiverInfo:  toModelSafeUser(pb.ReceiverInfo),
		CreatedAt:     fromTimestamp(pb.CreatedAt),
		UpdatedAt:     fromTimestamp(pb.UpdatedAt),
	}
}

func toModelSafeUser(pb *friendshippb.FriendshipUser) models.SafeUserResponse {
	if pb == nil {
		return models.SafeUserResponse{}
	}
	id, _ := primitive.ObjectIDFromHex(pb.Id)
	return models.SafeUserResponse{
		ID:       id,
		Username: pb.Username,
		FullName: pb.FullName,
		Avatar:   pb.Avatar,
	}
}

func toModelPopulatedFriendships(pbs []*friendshippb.PopulatedFriendship) []models.PopulatedFriendship {
	if len(pbs) == 0 {
		return []models.PopulatedFriendship{}
	}
	result := make([]models.PopulatedFriendship, 0, len(pbs))
	for _, pb := range pbs {
		if pf := toModelPopulatedFriendship(pb); pf != nil {
			result = append(result, *pf)
		}
	}
	return result
}

func toModelUser(pb *friendshippb.FriendshipUser) *models.User {
	if pb == nil {
		return nil
	}
	id, _ := primitive.ObjectIDFromHex(pb.Id)
	return &models.User{
		ID:       id,
		Username: pb.Username,
		FullName: pb.FullName,
		Avatar:   pb.Avatar,
	}
}

func toModelUsers(pbs []*friendshippb.FriendshipUser) []models.User {
	if len(pbs) == 0 {
		return []models.User{}
	}
	result := make([]models.User, 0, len(pbs))
	for _, pb := range pbs {
		if u := toModelUser(pb); u != nil {
			result = append(result, *u)
		}
	}
	return result
}

func toModelUserShort(pb *friendshippb.FriendshipUser) models.UserShortResponse {
	if pb == nil {
		return models.UserShortResponse{}
	}
	id, _ := primitive.ObjectIDFromHex(pb.Id)
	return models.UserShortResponse{
		ID:       id,
		Username: pb.Username,
		FullName: pb.FullName,
		Avatar:   pb.Avatar,
	}
}

func toModelUserShorts(pbs []*friendshippb.FriendshipUser) []models.UserShortResponse {
	if len(pbs) == 0 {
		return []models.UserShortResponse{}
	}
	result := make([]models.UserShortResponse, 0, len(pbs))
	for _, pb := range pbs {
		result = append(result, toModelUserShort(pb))
	}
	return result
}

// FriendshipStatusResponse represents detailed friendship status
type FriendshipStatusResponse struct {
	AreFriends        bool
	RequestSent       bool
	RequestReceived   bool
	IsBlockedByViewer bool
	HasBlockedViewer  bool
}

func toModelFriendshipStatus(pb *friendshippb.GetDetailedFriendshipStatusResponse) *FriendshipStatusResponse {
	if pb == nil {
		return nil
	}
	return &FriendshipStatusResponse{
		AreFriends:        pb.AreFriends,
		RequestSent:       pb.RequestSent,
		RequestReceived:   pb.RequestReceived,
		IsBlockedByViewer: pb.IsBlockedByViewer,
		HasBlockedViewer:  pb.HasBlockedViewer,
	}
}

// ============================================================================
// Model to Proto Converters
// ============================================================================

func toProtoFriendshipUser(u models.UserShortResponse) *friendshippb.FriendshipUser {
	return &friendshippb.FriendshipUser{
		Id:       u.ID.Hex(),
		Username: u.Username,
		FullName: u.FullName,
		Avatar:   u.Avatar,
	}
}

// ============================================================================
// Utility Converters
// ============================================================================

func fromTimestamp(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func hexOrEmpty(id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}
