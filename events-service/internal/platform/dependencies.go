package platform

import (
	"gitlab.com/spydotech-group/events-service/internal/cache"
	"gitlab.com/spydotech-group/events-service/internal/service"

	"gitlab.com/spydotech-group/events-service/internal/repository"
)

type repositoryBundle struct {
	Event *repository.EventRepository
	// UserLocal           *integration.UserLocalRepository // Needs to be added
	// FriendshipLocal     *integration.FriendshipReadOnlyRepository // Needs to be added
}

// Minimal buildRepositories
// simplified for now implies we fix the bootstrap flow in internal/bootstrap.go instead of here.
// But let's try to keep the structure valid.

type serviceBundle struct {
	Event               service.EventServiceContract
	EventRecommendation service.EventRecommendationServiceContract
	EventCache          *cache.EventCache
}
