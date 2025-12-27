package service

import (
	"context"
	"testing"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMarketplaceService_DeleteProduct_Authorization(t *testing.T) {
	tests := []struct {
		name         string
		productOwner primitive.ObjectID
		requesterID  primitive.ObjectID
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "owner can delete own product",
			productOwner: primitive.NewObjectID(),
			requesterID:  primitive.ObjectID{}, // Will be set to same as productOwner
			wantErr:      false,
		},
		{
			name:         "non-owner cannot delete product",
			productOwner: primitive.NewObjectID(),
			requesterID:  primitive.NewObjectID(),
			wantErr:      true,
			errMessage:   "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockMarketplaceRepository)
			businessMetrics := metrics.NewBusinessMetrics()
			service := NewMarketplaceService(mockRepo, businessMetrics, nil, nil)

			productID := primitive.NewObjectID()

			// Set requesterID to productOwner if empty (test case 1)
			if tt.requesterID.IsZero() {
				tt.requesterID = tt.productOwner
			}

			product := &models.Product{
				ID:       productID,
				SellerID: tt.productOwner,
				Title:    "Test Product",
				Status:   models.ProductStatusAvailable,
			}

			mockRepo.On("GetProductByID", mock.Anything, productID).Return(product, nil)

			if !tt.wantErr {
				mockRepo.On("DeleteProduct", mock.Anything, productID).Return(nil)
			}

			err := service.DeleteProduct(context.Background(), productID, tt.requesterID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMarketplaceService_ToggleSaveProduct(t *testing.T) {
	mockRepo := new(MockMarketplaceRepository)
	businessMetrics := metrics.NewBusinessMetrics()
	service := NewMarketplaceService(mockRepo, businessMetrics, nil, nil)

	productID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	tests := []struct {
		name           string
		savedBy        []primitive.ObjectID
		expectedSaved  bool
		expectedAction string
	}{
		{
			name:           "save product when not saved",
			savedBy:        []primitive.ObjectID{},
			expectedSaved:  true,
			expectedAction: "save",
		},
		{
			name:           "unsave product when already saved",
			savedBy:        []primitive.ObjectID{userID},
			expectedSaved:  false,
			expectedAction: "unsave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := &models.Product{
				ID:      productID,
				Title:   "Test Product",
				SavedBy: tt.savedBy,
			}

			mockRepo.On("GetProductByID", mock.Anything, productID).Return(product, nil)
			mockRepo.On("UpdateProduct", mock.Anything, productID, mock.Anything).Return(product, nil)

			saved, err := service.ToggleSaveProduct(context.Background(), productID, userID)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSaved, saved)
			mockRepo.AssertExpectations(t)

			// Reset for next iteration
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil
		})
	}
}
