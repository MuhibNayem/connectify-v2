package events

import (
	"time"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	eventspb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/events/v1"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoEventFromModel(event *models.Event) *eventspb.Event {
	if event == nil {
		return nil
	}

	return &eventspb.Event{
		Id:          event.ID.Hex(),
		Title:       event.Title,
		Description: event.Description,
		StartDate:   toTimestamp(event.StartDate),
		EndDate:     toTimestamp(event.EndDate),
		Location:    event.Location,
		IsOnline:    event.IsOnline,
		Privacy:     string(event.Privacy),
		Category:    event.Category,
		CoverImage:  event.CoverImage,
		Stats:       toProtoEventStats(event.Stats),
		CreatorId:   event.CreatorID.Hex(),
		Attendees:   toProtoAttendees(event.Attendees),
		CreatedAt:   toTimestamp(event.CreatedAt),
		UpdatedAt:   toTimestamp(event.UpdatedAt),
	}
}

func toProtoEventFromResponse(resp *models.EventResponse) *eventspb.Event {
	if resp == nil {
		return nil
	}

	ev := &eventspb.Event{
		Id:           resp.ID,
		Title:        resp.Title,
		Description:  resp.Description,
		StartDate:    toTimestamp(resp.StartDate),
		EndDate:      toTimestamp(resp.EndDate),
		Location:     resp.Location,
		IsOnline:     resp.IsOnline,
		Privacy:      string(resp.Privacy),
		Category:     resp.Category,
		CoverImage:   resp.CoverImage,
		Stats:        toProtoEventStats(resp.Stats),
		CreatedAt:    toTimestamp(resp.CreatedAt),
		MyStatus:     string(resp.MyStatus),
		IsHost:       resp.IsHost,
		FriendsGoing: toProtoUserShortSlice(resp.FriendsGoing),
	}

	if resp.Creator.ID != "" {
		ev.Creator = toProtoUserShort(resp.Creator)
		ev.CreatorId = resp.Creator.ID
	}
	return ev
}

func toProtoEventsFromResponses(events []models.EventResponse) []*eventspb.Event {
	result := make([]*eventspb.Event, 0, len(events))
	for i := range events {
		if event := toProtoEventFromResponse(&events[i]); event != nil {
			result = append(result, event)
		}
	}
	return result
}

func toProtoAttendees(attendees []models.EventAttendee) []*eventspb.EventAttendee {
	if len(attendees) == 0 {
		return nil
	}
	result := make([]*eventspb.EventAttendee, 0, len(attendees))
	for _, attendee := range attendees {
		result = append(result, &eventspb.EventAttendee{
			UserId:    attendee.UserID.Hex(),
			Status:    string(attendee.Status),
			Timestamp: toTimestamp(attendee.Timestamp),
		})
	}
	return result
}

func toProtoEventStats(stats models.EventStats) *eventspb.EventStats {
	return &eventspb.EventStats{
		GoingCount:      stats.GoingCount,
		InterestedCount: stats.InterestedCount,
		InvitedCount:    stats.InvitedCount,
		ShareCount:      stats.ShareCount,
	}
}

func toProtoUserShort(user models.UserShort) *eventspb.UserShort {
	if user.ID == "" && user.Username == "" && user.FullName == "" && user.Avatar == "" {
		return nil
	}
	return &eventspb.UserShort{
		Id:       user.ID,
		Username: user.Username,
		FullName: user.FullName,
		Avatar:   user.Avatar,
	}
}

func toProtoUserShortSlice(users []models.UserShort) []*eventspb.UserShort {
	if len(users) == 0 {
		return nil
	}
	result := make([]*eventspb.UserShort, 0, len(users))
	for _, user := range users {
		if u := toProtoUserShort(user); u != nil {
			result = append(result, u)
		}
	}
	return result
}

func toProtoBirthdays(response *models.BirthdayResponse) *eventspb.BirthdaysResponse {
	if response == nil {
		return &eventspb.BirthdaysResponse{}
	}
	return &eventspb.BirthdaysResponse{
		Today:    toProtoBirthdayUsers(response.Today),
		Upcoming: toProtoBirthdayUsers(response.Upcoming),
	}
}

func toProtoBirthdayUsers(users []models.BirthdayUser) []*eventspb.BirthdayUser {
	if len(users) == 0 {
		return nil
	}
	result := make([]*eventspb.BirthdayUser, 0, len(users))
	for _, user := range users {
		result = append(result, &eventspb.BirthdayUser{
			Id:       user.ID,
			Username: user.Username,
			FullName: user.FullName,
			Avatar:   user.Avatar,
			Age:      int32(user.Age),
			Date:     user.Date,
		})
	}
	return result
}

func toProtoInvitation(inv models.EventInvitationResponse) *eventspb.Invitation {
	return &eventspb.Invitation{
		Id:        inv.ID,
		Event:     toProtoEventShort(inv.Event),
		Inviter:   toProtoUserShort(inv.Inviter),
		Status:    string(inv.Status),
		Message:   inv.Message,
		CreatedAt: toTimestamp(inv.CreatedAt),
	}
}

func toProtoEventShort(short models.EventShort) *eventspb.EventShort {
	return &eventspb.EventShort{
		Id:         short.ID,
		Title:      short.Title,
		CoverImage: short.CoverImage,
		StartDate:  toTimestamp(short.StartDate),
		Location:   short.Location,
	}
}

func toProtoPost(post *models.EventPostResponse) *eventspb.EventPost {
	if post == nil {
		return nil
	}

	return &eventspb.EventPost{
		Id:        post.ID,
		Content:   post.Content,
		MediaUrls: append([]string(nil), post.MediaURLs...),
		CreatedAt: toTimestamp(post.CreatedAt),
		Author:    toProtoUserShort(post.Author),
		Reactions: toProtoPostReactions(post.Reactions),
	}
}

func toProtoPostReactions(reactions []models.EventPostReactionResponse) []*eventspb.EventPostReaction {
	if len(reactions) == 0 {
		return nil
	}
	result := make([]*eventspb.EventPostReaction, 0, len(reactions))
	for _, reaction := range reactions {
		result = append(result, &eventspb.EventPostReaction{
			User:      toProtoUserShort(reaction.User),
			Emoji:     reaction.Emoji,
			Timestamp: toTimestamp(reaction.Timestamp),
		})
	}
	return result
}

func toProtoAttendeeList(resp *models.AttendeesListResponse) *eventspb.GetAttendeesResponse {
	if resp == nil {
		return &eventspb.GetAttendeesResponse{}
	}

	result := &eventspb.GetAttendeesResponse{
		Attendees: make([]*eventspb.EventAttendeeView, 0, len(resp.Attendees)),
		Total:     resp.Total,
		Page:      resp.Page,
		Limit:     resp.Limit,
	}
	for _, attendee := range resp.Attendees {
		result.Attendees = append(result.Attendees, &eventspb.EventAttendeeView{
			User:      toProtoUserShort(attendee.User),
			Status:    string(attendee.Status),
			Timestamp: toTimestamp(attendee.Timestamp),
			IsHost:    attendee.IsHost,
			IsCohost:  attendee.IsCoHost,
		})
	}
	return result
}

func toProtoCategories(categories []models.EventCategory) []*eventspb.EventCategory {
	if len(categories) == 0 {
		return nil
	}
	result := make([]*eventspb.EventCategory, 0, len(categories))
	for _, category := range categories {
		result = append(result, &eventspb.EventCategory{
			Name:  category.Name,
			Icon:  category.Icon,
			Count: category.Count,
		})
	}
	return result
}

func toProtoRecommendations(recs []service.EventRecommendation) []*eventspb.Recommendation {
	if len(recs) == 0 {
		return nil
	}
	result := make([]*eventspb.Recommendation, 0, len(recs))
	for _, rec := range recs {
		var protoEvent *eventspb.Event
		if rec.Event != nil {
			protoEvent = toProtoEventFromModel(rec.Event)
		}
		result = append(result, &eventspb.Recommendation{
			Event:        protoEvent,
			FriendsGoing: toProtoUserShortSlice(rec.FriendsGoing),
			Reason:       rec.Reason,
			Score:        rec.Score,
		})
	}
	return result
}

func toProtoTrending(scores []service.TrendingScore) []*eventspb.TrendingScore {
	if len(scores) == 0 {
		return nil
	}
	result := make([]*eventspb.TrendingScore, 0, len(scores))
	for _, score := range scores {
		var protoEvent *eventspb.Event
		if score.Event != nil {
			protoEvent = toProtoEventFromModel(score.Event)
		}
		result = append(result, &eventspb.TrendingScore{
			Event: protoEvent,
			Score: score.Score,
		})
	}
	return result
}

func toTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func objectIDFromString(value string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(value)
}
