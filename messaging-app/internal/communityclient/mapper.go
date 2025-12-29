package communityclient

import (
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	communitypb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/community/v1"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func hexOrEmpty(id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}

func toModelCommunity(pb *communitypb.Community) *models.Community {
	if pb == nil {
		return nil
	}
	id, _ := primitive.ObjectIDFromHex(pb.Id)
	creatorID, _ := primitive.ObjectIDFromHex(pb.CreatorId)

	members := make([]primitive.ObjectID, 0, len(pb.Members))
	for _, m := range pb.Members {
		if oid, err := primitive.ObjectIDFromHex(m); err == nil {
			members = append(members, oid)
		}
	}
	admins := make([]primitive.ObjectID, 0, len(pb.Admins))
	for _, a := range pb.Admins {
		if oid, err := primitive.ObjectIDFromHex(a); err == nil {
			admins = append(admins, oid)
		}
	}
	pending := make([]primitive.ObjectID, 0, len(pb.PendingMembers))
	for _, p := range pb.PendingMembers {
		if oid, err := primitive.ObjectIDFromHex(p); err == nil {
			pending = append(pending, oid)
		}
	}
	banned := make([]primitive.ObjectID, 0, len(pb.BannedUsers))
	for _, b := range pb.BannedUsers {
		if oid, err := primitive.ObjectIDFromHex(b); err == nil {
			banned = append(banned, oid)
		}
	}

	rules := make([]models.CommunityRule, len(pb.Rules))
	if pb.Rules != nil {
		for i, r := range pb.Rules {
			rules[i] = models.CommunityRule{Title: r.Title, Description: r.Description}
		}
	}

	settings := models.CommunitySettings{}
	if pb.Settings != nil {
		settings = models.CommunitySettings{
			RequirePostApproval:  pb.Settings.RequirePostApproval,
			RequireJoinApproval:  pb.Settings.RequireJoinApproval,
			AllowMemberPosts:     pb.Settings.AllowMemberPosts,
			ShowGroupAffiliation: pb.Settings.ShowGroupAffiliation,
		}
	}

	stats := models.CommunityStats{}
	if pb.Stats != nil {
		stats = models.CommunityStats{
			MemberCount: pb.Stats.MemberCount,
			PostCount:   pb.Stats.PostCount,
		}
	}

	return &models.Community{
		ID:                  id,
		Name:                pb.Name,
		Description:         pb.Description,
		Slug:                pb.Slug,
		Category:            pb.Category,
		Avatar:              pb.Avatar,
		CoverImage:          pb.CoverImage,
		Privacy:             models.CommunityPrivacy(pb.Privacy),
		Visibility:          models.CommunityVisibility(pb.Visibility),
		CreatorID:           creatorID,
		Members:             members,
		Admins:              admins,
		PendingMembers:      pending,
		BannedUsers:         banned,
		Settings:            settings,
		Rules:               rules,
		MembershipQuestions: pb.MembershipQuestions,
		Stats:               stats,
		CreatedAt:           pb.CreatedAt.AsTime(),
		UpdatedAt:           pb.UpdatedAt.AsTime(),
	}
}

func toModelCommunityResponse(pb *communitypb.Community) *models.CommunityResponse {
	if pb == nil {
		return nil
	}

	c := toModelCommunity(pb)

	return &models.CommunityResponse{
		ID:                  c.ID.Hex(),
		Name:                c.Name,
		Description:         c.Description,
		Slug:                c.Slug,
		Category:            c.Category,
		Avatar:              c.Avatar,
		CoverImage:          c.CoverImage,
		Privacy:             c.Privacy,
		Visibility:          c.Visibility,
		Settings:            c.Settings,
		Rules:               c.Rules,
		MembershipQuestions: c.MembershipQuestions,
		Stats:               c.Stats,
		IsMember:            pb.IsMember,
		IsAdmin:             pb.IsAdmin,
		IsPending:           pb.IsPending,
		CreatedAt:           c.CreatedAt,
	}
}

func toProtoCreateCommunityRequest(req models.CreateCommunityRequest, creatorID primitive.ObjectID) *communitypb.CreateCommunityRequest {
	return &communitypb.CreateCommunityRequest{
		CreatorId:           creatorID.Hex(),
		Name:                req.Name,
		Description:         req.Description,
		Category:            req.Category,
		Avatar:              req.Avatar,
		CoverImage:          req.CoverImage,
		Privacy:             string(req.Privacy),
		Visibility:          string(req.Visibility),
		RequirePostApproval: req.RequirePostApproval,
		RequireJoinApproval: req.RequireJoinApproval,
	}
}

func toProtoUpdateCommunityRequest(req models.UpdateCommunityRequest, communityID, userID primitive.ObjectID) *communitypb.UpdateCommunityRequest {
	pbReq := &communitypb.UpdateCommunityRequest{
		CommunityId: communityID.Hex(),
		UserId:      userID.Hex(),
	}
	if req.Name != "" {
		pbReq.Name = &req.Name
	}
	if req.Description != "" {
		pbReq.Description = &req.Description
	}
	if req.Category != "" {
		pbReq.Category = &req.Category
	}
	if req.Avatar != "" {
		pbReq.Avatar = &req.Avatar
	}
	if req.CoverImage != "" {
		pbReq.CoverImage = &req.CoverImage
	}
	if req.Privacy != "" {
		p := string(req.Privacy)
		pbReq.Privacy = &p
	}
	if req.Visibility != "" {
		v := string(req.Visibility)
		pbReq.Visibility = &v
	}
	if req.RequirePostApproval != nil {
		pbReq.RequirePostApproval = req.RequirePostApproval
	}
	if req.RequireJoinApproval != nil {
		pbReq.RequireJoinApproval = req.RequireJoinApproval
	}
	if req.AllowMemberPosts != nil {
		pbReq.AllowMemberPosts = req.AllowMemberPosts
	}
	if req.ShowGroupAffiliation != nil {
		pbReq.ShowGroupAffiliation = req.ShowGroupAffiliation
	}

	if len(req.Rules) > 0 {
		rules := make([]*communitypb.CommunityRule, len(req.Rules))
		for i, r := range req.Rules {
			rules[i] = &communitypb.CommunityRule{Title: r.Title, Description: r.Description}
		}
		pbReq.Rules = rules
	}
	if len(req.MembershipQuestions) > 0 {
		pbReq.MembershipQuestions = req.MembershipQuestions
	}
	return pbReq
}

func toModelUser(pb *communitypb.UserShort) models.User {
	if pb == nil {
		return models.User{}
	}
	id, _ := primitive.ObjectIDFromHex(pb.Id)
	return models.User{
		ID:       id,
		Username: pb.Username,
		FullName: pb.FullName,
		Avatar:   pb.Avatar,
	}
}

func toModelUsers(pbs []*communitypb.UserShort) []models.User {
	if pbs == nil {
		return nil
	}
	users := make([]models.User, len(pbs))
	for i, pb := range pbs {
		users[i] = toModelUser(pb)
	}
	return users
}
