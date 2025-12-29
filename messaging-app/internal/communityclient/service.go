package communityclient

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	communitypb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/community/v1"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (c *Client) CreateCommunity(ctx context.Context, userID primitive.ObjectID, req models.CreateCommunityRequest) (*models.Community, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.CreateCommunity(ctx, toProtoCreateCommunityRequest(req, userID))
	})
	if err != nil {
		return nil, err
	}
	return toModelCommunity(result.(*communitypb.CreateCommunityResponse).Community), nil
}

func (c *Client) GetCommunity(ctx context.Context, communityID primitive.ObjectID, viewerID primitive.ObjectID) (*models.CommunityResponse, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetCommunity(ctx, &communitypb.GetCommunityRequest{
			CommunityId: communityID.Hex(),
			ViewerId:    hexOrEmpty(viewerID),
		})
	})
	if err != nil {
		return nil, err
	}
	return toModelCommunityResponse(result.(*communitypb.GetCommunityResponse).Community), nil
}

func (c *Client) UpdateCommunity(ctx context.Context, communityID, userID primitive.ObjectID, req models.UpdateCommunityRequest) (*models.Community, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UpdateCommunity(ctx, toProtoUpdateCommunityRequest(req, communityID, userID))
	})
	if err != nil {
		return nil, err
	}
	return toModelCommunity(result.(*communitypb.UpdateCommunityResponse).Community), nil
}

func (c *Client) ListCommunities(ctx context.Context, viewerID primitive.ObjectID, limit, page int64, query string) ([]models.CommunityResponse, int64, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ListCommunities(ctx, &communitypb.ListCommunitiesRequest{
			Query:    query,
			Limit:    limit,
			Page:     page,
			ViewerId: hexOrEmpty(viewerID),
		})
	})
	if err != nil {
		return nil, 0, err
	}

	resp := result.(*communitypb.ListCommunitiesResponse)
	communities := make([]models.CommunityResponse, len(resp.Communities))
	for i, pb := range resp.Communities {
		communities[i] = *toModelCommunityResponse(pb)
	}

	return communities, resp.Total, nil
}

func (c *Client) GetUserCommunities(ctx context.Context, userID primitive.ObjectID) ([]models.Community, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetUserCommunities(ctx, &communitypb.GetUserCommunitiesRequest{UserId: userID.Hex()})
	})
	if err != nil {
		return nil, err
	}

	resp := result.(*communitypb.GetUserCommunitiesResponse)
	communities := make([]models.Community, len(resp.Communities))
	for i, pb := range resp.Communities {
		communities[i] = *toModelCommunity(pb)
	}

	return communities, nil
}

func (c *Client) JoinCommunity(ctx context.Context, communityID, userID primitive.ObjectID) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.JoinCommunity(ctx, &communitypb.JoinCommunityRequest{
			CommunityId: communityID.Hex(),
			UserId:      userID.Hex(),
		})
	})
	return err
}

func (c *Client) LeaveCommunity(ctx context.Context, communityID, userID primitive.ObjectID) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.LeaveCommunity(ctx, &communitypb.LeaveCommunityRequest{
			CommunityId: communityID.Hex(),
			UserId:      userID.Hex(),
		})
	})
	return err
}

func (c *Client) ApproveMember(ctx context.Context, communityID, actorID, targetID primitive.ObjectID) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.ApproveMember(ctx, &communitypb.ApproveMemberRequest{
			CommunityId: communityID.Hex(),
			ActorId:     actorID.Hex(),
			TargetId:    targetID.Hex(),
		})
	})
	return err
}

func (c *Client) RejectMember(ctx context.Context, communityID, actorID, targetID primitive.ObjectID) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.RejectMember(ctx, &communitypb.RejectMemberRequest{
			CommunityId: communityID.Hex(),
			ActorId:     actorID.Hex(),
			TargetId:    targetID.Hex(),
		})
	})
	return err
}

func (c *Client) GetMembers(ctx context.Context, communityID, viewerID primitive.ObjectID, limit, page int64) ([]models.User, int64, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetMembers(ctx, &communitypb.GetMembersRequest{
			CommunityId: communityID.Hex(),
			ViewerId:    hexOrEmpty(viewerID),
			Limit:       limit,
			Page:        page,
		})
	})
	if err != nil {
		return nil, 0, err
	}
	resp := result.(*communitypb.GetMembersResponse)
	return toModelUsers(resp.Members), resp.Total, nil
}

func (c *Client) GetAdmins(ctx context.Context, communityID primitive.ObjectID) ([]models.User, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetAdmins(ctx, &communitypb.GetAdminsRequest{
			CommunityId: communityID.Hex(),
		})
	})
	if err != nil {
		return nil, err
	}
	resp := result.(*communitypb.GetAdminsResponse)
	return toModelUsers(resp.Admins), nil
}

func (c *Client) GetPendingMembers(ctx context.Context, communityID, actorID primitive.ObjectID, limit, page int64) ([]models.User, int64, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetPendingMembers(ctx, &communitypb.GetPendingMembersRequest{
			CommunityId: communityID.Hex(),
			ActorId:     actorID.Hex(),
			Limit:       limit,
			Page:        page,
		})
	})
	if err != nil {
		return nil, 0, err
	}
	resp := result.(*communitypb.GetPendingMembersResponse)
	return toModelUsers(resp.Members), resp.Total, nil
}
