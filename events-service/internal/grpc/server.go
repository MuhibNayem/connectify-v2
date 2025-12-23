package grpc

import (
	"context"

	"gitlab.com/spydotech-group/events-service/internal/service"
	"gitlab.com/spydotech-group/shared-entity/models"
	eventspb "gitlab.com/spydotech-group/shared-entity/proto/events/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	eventspb.UnimplementedEventsServiceServer
	eventService          *service.EventService
	recommendationService *service.EventRecommendationService
}

func NewServer(eventService *service.EventService, recommendationService *service.EventRecommendationService) *Server {
	return &Server{
		eventService:          eventService,
		recommendationService: recommendationService,
	}
}

func (s *Server) CreateEvent(ctx context.Context, req *eventspb.CreateEventRequest) (*eventspb.EventResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	modelReq := models.CreateEventRequest{
		Title:       req.Event.Title,
		Description: req.Event.Description,
		StartDate:   req.Event.StartDate.AsTime(),
		EndDate:     req.Event.EndDate.AsTime(),
		Location:    req.Event.Location,
		Privacy:     models.EventPrivacy(req.Event.Privacy),
		Category:    req.Event.Category,
		CoverImage:  req.Event.CoverImage,
	}

	if req.Event.IsOnline != nil {
		modelReq.IsOnline = *req.Event.IsOnline
	}

	event, err := s.eventService.CreateEvent(ctx, userID, modelReq)
	if err != nil {
		return nil, err
	}

	return &eventspb.EventResponse{Event: toProtoEvent(event)}, nil
}

func (s *Server) GetEvent(ctx context.Context, req *eventspb.GetEventRequest) (*eventspb.EventResponse, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}

	var viewerID primitive.ObjectID
	if req.ViewerId != "" {
		viewerID, _ = primitive.ObjectIDFromHex(req.ViewerId)
	}

	event, err := s.eventService.GetEvent(ctx, eventID, viewerID)
	if err != nil {
		return nil, err
	}

	return &eventspb.EventResponse{Event: toProtoEventFromResponse(event)}, nil
}

func (s *Server) UpdateEvent(ctx context.Context, req *eventspb.UpdateEventRequest) (*eventspb.EventResponse, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	start := req.Event.StartDate.AsTime()
	end := req.Event.EndDate.AsTime()

	modelReq := models.UpdateEventRequest{
		Title:       req.Event.Title,
		Description: req.Event.Description,
		StartDate:   &start,
		EndDate:     &end,
		Location:    req.Event.Location,
		Privacy:     models.EventPrivacy(req.Event.Privacy),
		Category:    req.Event.Category,
		CoverImage:  req.Event.CoverImage,
	}
	if req.Event.IsOnline != nil {
		modelReq.IsOnline = req.Event.IsOnline
	}

	event, err := s.eventService.UpdateEvent(ctx, eventID, userID, modelReq)
	if err != nil {
		return nil, err
	}

	return &eventspb.EventResponse{Event: toProtoEventFromResponse(event)}, nil
}

func (s *Server) DeleteEvent(ctx context.Context, req *eventspb.DeleteEventRequest) (*emptypb.Empty, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	if err := s.eventService.DeleteEvent(ctx, eventID, userID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) ListEvents(ctx context.Context, req *eventspb.ListEventsRequest) (*eventspb.ListEventsResponse, error) {
	var userID primitive.ObjectID
	if req.UserId != "" {
		userID, _ = primitive.ObjectIDFromHex(req.UserId)
	}

	events, total, err := s.eventService.ListEvents(ctx, userID, req.Limit, req.Page, req.Query, req.Category, req.Period)
	if err != nil {
		return nil, err
	}

	return &eventspb.ListEventsResponse{
		Events: toProtoEventsFromResponses(events),
		Total:  total,
		Page:   req.Page,
		Limit:  req.Limit,
	}, nil
}

func (s *Server) RSVP(ctx context.Context, req *eventspb.RSVPRequest) (*emptypb.Empty, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	if err := s.eventService.RSVP(ctx, eventID, userID, models.RSVPStatus(req.Status)); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetMyEvents(ctx context.Context, req *eventspb.GetMyEventsRequest) (*eventspb.ListEventsResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	events, err := s.eventService.GetUserEvents(ctx, userID, req.Limit, req.Page)
	if err != nil {
		return nil, err
	}
	return &eventspb.ListEventsResponse{Events: toProtoEventsFromResponses(events)}, nil
}

func (s *Server) GetBirthdays(ctx context.Context, req *eventspb.GetBirthdaysRequest) (*eventspb.BirthdaysResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	response, err := s.eventService.GetFriendBirthdays(ctx, userID)
	if err != nil {
		return nil, err
	}

	toProtoUsers := func(users []models.BirthdayUser) []*eventspb.BirthdayUser {
		var res []*eventspb.BirthdayUser
		for _, u := range users {
			res = append(res, &eventspb.BirthdayUser{
				Id:       u.ID,
				Username: u.Username,
				FullName: u.FullName,
				Avatar:   u.Avatar,
				Age:      int32(u.Age),
				Date:     u.Date,
			})
		}
		return res
	}

	return &eventspb.BirthdaysResponse{
		Today:    toProtoUsers(response.Today),
		Upcoming: toProtoUsers(response.Upcoming),
	}, nil
}

func (s *Server) InviteFriends(ctx context.Context, req *eventspb.InviteFriendsRequest) (*emptypb.Empty, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	inviterID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	if err := s.eventService.InviteFriends(ctx, eventID, inviterID, req.FriendIds, req.Message); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetInvitations(ctx context.Context, req *eventspb.GetInvitationsRequest) (*eventspb.GetInvitationsResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	invitations, total, err := s.eventService.GetUserInvitations(ctx, userID, req.Limit, req.Page)
	if err != nil {
		return nil, err
	}

	var pbInvitations []*eventspb.Invitation
	for _, inv := range invitations {
		pbInvitations = append(pbInvitations, &eventspb.Invitation{
			Id: inv.ID,
			Event: &eventspb.EventShort{
				Id:         inv.Event.ID,
				Title:      inv.Event.Title,
				CoverImage: inv.Event.CoverImage,
				StartDate:  timestamppb.New(inv.Event.StartDate),
				Location:   inv.Event.Location,
			},
			Inviter: &eventspb.UserShort{
				Id:       inv.Inviter.ID,
				Username: inv.Inviter.Username,
				FullName: inv.Inviter.FullName,
				Avatar:   inv.Inviter.Avatar,
			},
			Status:    string(inv.Status),
			Message:   inv.Message,
			CreatedAt: timestamppb.New(inv.CreatedAt),
		})
	}

	return &eventspb.GetInvitationsResponse{
		Invitations: pbInvitations,
		Total:       total,
		Page:        req.Page,
		Limit:       req.Limit,
	}, nil
}

func (s *Server) RespondToInvitation(ctx context.Context, req *eventspb.RespondToInvitationRequest) (*emptypb.Empty, error) {
	invitationID, err := primitive.ObjectIDFromHex(req.InvitationId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	if err := s.eventService.RespondToInvitation(ctx, invitationID, userID, req.Accept); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) CreatePost(ctx context.Context, req *eventspb.CreatePostRequest) (*eventspb.EventPost, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	authorID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	resp, err := s.eventService.CreatePost(ctx, eventID, authorID, models.CreateEventPostRequest{
		Content:   req.Content,
		MediaURLs: req.MediaUrls,
	})
	if err != nil {
		return nil, err
	}

	return toProtoEventPost(resp), nil
}

func (s *Server) GetPosts(ctx context.Context, req *eventspb.GetPostsRequest) (*eventspb.GetPostsResponse, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	posts, total, err := s.eventService.GetPosts(ctx, eventID, req.Limit, req.Page)
	if err != nil {
		return nil, err
	}

	var pbPosts []*eventspb.EventPost
	for _, p := range posts {
		pbPosts = append(pbPosts, toProtoEventPost(&p))
	}

	return &eventspb.GetPostsResponse{
		Posts: pbPosts,
		Total: total,
		Page:  req.Page,
		Limit: req.Limit,
	}, nil
}

func (s *Server) DeletePost(ctx context.Context, req *eventspb.DeletePostRequest) (*emptypb.Empty, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	postID, err := primitive.ObjectIDFromHex(req.PostId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	if err := s.eventService.DeletePost(ctx, eventID, postID, userID); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) ReactToPost(ctx context.Context, req *eventspb.ReactToPostRequest) (*emptypb.Empty, error) {
	postID, err := primitive.ObjectIDFromHex(req.PostId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	// Proto field is "Reaction", service method takes emoji string
	if err := s.eventService.ReactToPost(ctx, postID, userID, req.Reaction); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetAttendees(ctx context.Context, req *eventspb.GetAttendeesRequest) (*eventspb.GetAttendeesResponse, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	resp, err := s.eventService.GetAttendees(ctx, eventID, models.RSVPStatus(req.Status), req.Limit, req.Page)
	if err != nil {
		return nil, err
	}

	var pbAttendees []*eventspb.EventAttendeeView
	for _, a := range resp.Attendees {
		pbAttendees = append(pbAttendees, &eventspb.EventAttendeeView{
			User: &eventspb.UserShort{
				Id:       a.User.ID,
				Username: a.User.Username,
				FullName: a.User.FullName,
				Avatar:   a.User.Avatar,
			},
			Status:    string(a.Status),
			Timestamp: timestamppb.New(a.Timestamp),
			IsHost:    a.IsHost,
			IsCohost:  a.IsCoHost,
		})
	}

	return &eventspb.GetAttendeesResponse{
		Attendees: pbAttendees,
		Total:     resp.Total,
		Page:      resp.Page,
		Limit:     resp.Limit,
	}, nil
}

func (s *Server) AddCoHost(ctx context.Context, req *eventspb.CoHostRequest) (*emptypb.Empty, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	coHostID, err := primitive.ObjectIDFromHex(req.TargetUserId)
	if err != nil {
		return nil, err
	}
	if err := s.eventService.AddCoHost(ctx, eventID, userID, coHostID); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) RemoveCoHost(ctx context.Context, req *eventspb.CoHostRequest) (*emptypb.Empty, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}
	coHostID, err := primitive.ObjectIDFromHex(req.TargetUserId)
	if err != nil {
		return nil, err
	}
	if err := s.eventService.RemoveCoHost(ctx, eventID, userID, coHostID); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetCategories(ctx context.Context, req *emptypb.Empty) (*eventspb.GetCategoriesResponse, error) {
	cats, err := s.eventService.GetCategories(ctx)
	if err != nil {
		return nil, err
	}
	var pbCats []*eventspb.EventCategory
	for _, c := range cats {
		pbCats = append(pbCats, &eventspb.EventCategory{
			Name:  c.Name,
			Icon:  c.Icon,
			Count: c.Count,
		})
	}
	return &eventspb.GetCategoriesResponse{Categories: pbCats}, nil
}

func (s *Server) SearchEvents(ctx context.Context, req *eventspb.SearchEventsRequest) (*eventspb.ListEventsResponse, error) {
	var userID primitive.ObjectID
	if req.UserId != "" {
		userID, _ = primitive.ObjectIDFromHex(req.UserId)
	}

	// Map proto fields to model fields
	searchReq := models.SearchEventsRequest{
		Query:    req.Query,
		Category: req.Category,
		Period:   req.Timeframe, // Proto uses Timeframe, model uses Period
		Limit:    req.Limit,
		Page:     req.Page,
	}
	if req.IsOnline != nil {
		searchReq.Online = req.IsOnline
	}

	events, total, err := s.eventService.SearchEvents(ctx, searchReq, userID)
	if err != nil {
		return nil, err
	}

	return &eventspb.ListEventsResponse{
		Events: toProtoEventsFromResponses(events),
		Total:  total,
		Page:   req.Page,
		Limit:  req.Limit,
	}, nil
}

func (s *Server) ShareEvent(ctx context.Context, req *eventspb.ShareEventRequest) (*emptypb.Empty, error) {
	eventID, err := primitive.ObjectIDFromHex(req.EventId)
	if err != nil {
		return nil, err
	}
	if err := s.eventService.ShareEvent(ctx, eventID); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetNearbyEvents(ctx context.Context, req *eventspb.NearbyEventsRequest) (*eventspb.ListEventsResponse, error) {
	var userID primitive.ObjectID
	if req.UserId != "" {
		userID, _ = primitive.ObjectIDFromHex(req.UserId)
	}

	// Proto uses Latitude, Longitude, RadiusKm
	events, total, err := s.eventService.GetNearbyEvents(ctx, req.Latitude, req.Longitude, req.RadiusKm, req.Limit, req.Page, userID)
	if err != nil {
		return nil, err
	}

	return &eventspb.ListEventsResponse{
		Events: toProtoEventsFromResponses(events),
		Total:  total,
		Page:   req.Page,
		Limit:  req.Limit,
	}, nil
}

func (s *Server) GetRecommendations(ctx context.Context, req *eventspb.RecommendationRequest) (*eventspb.RecommendationResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, err
	}

	limit := 10
	if req.Limit > 0 {
		limit = int(req.Limit)
	}

	recs, err := s.recommendationService.GetRecommendations(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	// Proto type is Recommendation, not EventRecommendation
	var pbRecs []*eventspb.Recommendation
	for _, r := range recs {
		var friends []*eventspb.UserShort
		for _, f := range r.FriendsGoing {
			friends = append(friends, &eventspb.UserShort{
				Id:       f.ID,
				Username: f.Username,
				Avatar:   f.Avatar,
			})
		}

		var pbEvent *eventspb.Event
		if r.Event != nil {
			pbEvent = toProtoEvent(r.Event)
		}

		pbRecs = append(pbRecs, &eventspb.Recommendation{
			Event:        pbEvent,
			FriendsGoing: friends,
			Reason:       r.Reason,
			Score:        r.Score,
		})
	}

	return &eventspb.RecommendationResponse{Recommendations: pbRecs}, nil
}

func (s *Server) GetTrending(ctx context.Context, req *eventspb.TrendingEventsRequest) (*eventspb.TrendingEventsResponse, error) {
	limit := 10
	if req.Limit > 0 {
		limit = int(req.Limit)
	}

	trending, err := s.recommendationService.GetTrendingEvents(ctx, limit)
	if err != nil {
		return nil, err
	}

	// Proto type is TrendingScore
	var pbTrending []*eventspb.TrendingScore
	for _, t := range trending {
		var pbEvent *eventspb.Event
		if t.Event != nil {
			pbEvent = toProtoEvent(t.Event)
		}
		pbTrending = append(pbTrending, &eventspb.TrendingScore{
			Event: pbEvent,
			Score: t.Score,
		})
	}

	return &eventspb.TrendingEventsResponse{Trending: pbTrending}, nil
}

func (s *Server) ReportRSVPEvent(ctx context.Context, req *eventspb.ReportRSVPEventRequest) (*emptypb.Empty, error) {
	// Legacy compatibility stub
	return &emptypb.Empty{}, nil
}

// ===============================
// Helper Functions
// ===============================

func toProtoEvent(e *models.Event) *eventspb.Event {
	var startDate, endDate, createdAt, updatedAt *timestamppb.Timestamp
	if !e.StartDate.IsZero() {
		startDate = timestamppb.New(e.StartDate)
	}
	if !e.EndDate.IsZero() {
		endDate = timestamppb.New(e.EndDate)
	}
	if !e.CreatedAt.IsZero() {
		createdAt = timestamppb.New(e.CreatedAt)
	}
	if !e.UpdatedAt.IsZero() {
		updatedAt = timestamppb.New(e.UpdatedAt)
	}

	return &eventspb.Event{
		Id:          e.ID.Hex(),
		Title:       e.Title,
		Description: e.Description,
		StartDate:   startDate,
		EndDate:     endDate,
		Location:    e.Location,
		IsOnline:    e.IsOnline,
		Privacy:     string(e.Privacy),
		Category:    e.Category,
		CoverImage:  e.CoverImage,
		CreatorId:   e.CreatorID.Hex(),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func toProtoEventFromResponse(e *models.EventResponse) *eventspb.Event {
	var startDate, endDate, createdAt *timestamppb.Timestamp
	if !e.StartDate.IsZero() {
		startDate = timestamppb.New(e.StartDate)
	}
	if !e.EndDate.IsZero() {
		endDate = timestamppb.New(e.EndDate)
	}
	if !e.CreatedAt.IsZero() {
		createdAt = timestamppb.New(e.CreatedAt)
	}

	var friendsGoing []*eventspb.UserShort
	for _, f := range e.FriendsGoing {
		friendsGoing = append(friendsGoing, &eventspb.UserShort{
			Id:       f.ID,
			Username: f.Username,
			FullName: f.FullName,
			Avatar:   f.Avatar,
		})
	}

	return &eventspb.Event{
		Id:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		StartDate:   startDate,
		EndDate:     endDate,
		Location:    e.Location,
		IsOnline:    e.IsOnline,
		Privacy:     string(e.Privacy),
		Category:    e.Category,
		CoverImage:  e.CoverImage,
		Creator: &eventspb.UserShort{
			Id:       e.Creator.ID,
			Username: e.Creator.Username,
			FullName: e.Creator.FullName,
			Avatar:   e.Creator.Avatar,
		},
		MyStatus:     string(e.MyStatus),
		IsHost:       e.IsHost,
		FriendsGoing: friendsGoing,
		CreatedAt:    createdAt,
	}
}

func toProtoEventsFromResponses(es []models.EventResponse) []*eventspb.Event {
	var res []*eventspb.Event
	for _, e := range es {
		res = append(res, toProtoEventFromResponse(&e))
	}
	return res
}

func toProtoEventPost(p *models.EventPostResponse) *eventspb.EventPost {
	var reactions []*eventspb.EventPostReaction
	for _, r := range p.Reactions {
		reactions = append(reactions, &eventspb.EventPostReaction{
			User: &eventspb.UserShort{
				Id:       r.User.ID,
				Username: r.User.Username,
				Avatar:   r.User.Avatar,
			},
			Emoji:     r.Emoji,
			Timestamp: timestamppb.New(r.Timestamp),
		})
	}
	return &eventspb.EventPost{
		Id: p.ID,
		Author: &eventspb.UserShort{
			Id:       p.Author.ID,
			Username: p.Author.Username,
			FullName: p.Author.FullName,
			Avatar:   p.Author.Avatar,
		},
		Content:   p.Content,
		MediaUrls: p.MediaURLs,
		Reactions: reactions,
		CreatedAt: timestamppb.New(p.CreatedAt),
	}
}
