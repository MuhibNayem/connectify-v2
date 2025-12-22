package userclient

import (
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
	return &models.User{
		ID:       oid,
		Username: u.Username,
		Email:    u.Email,
		FullName: u.FullName,
		Avatar:   u.Avatar,
		Bio:      u.Bio,
	}, nil
}
