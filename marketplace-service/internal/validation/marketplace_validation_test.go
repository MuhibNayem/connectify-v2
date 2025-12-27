package validation

import (
	"testing"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
)

func TestValidateCreateProductRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         models.CreateProductRequest
		wantErr     bool
		expectedErr error
	}{
		{
			name: "valid request",
			req: models.CreateProductRequest{
				Title:       "iPhone 15 Pro",
				Description: "Brand new iPhone 15 Pro in excellent condition",
				Price:       999.99,
				Currency:    "USD",
				Images:      []string{"https://example.com/image1.jpg"},
				Location:    "New York, NY",
				CategoryID:  "507f1f77bcf86cd799439011",
				Tags:        []string{"phone", "apple"},
			},
			wantErr: false,
		},
		{
			name: "empty title",
			req: models.CreateProductRequest{
				Title:       "",
				Description: "Description",
				Price:       100,
				Currency:    "USD",
				Images:      []string{"https://example.com/image1.jpg"},
				Location:    "New York, NY",
				CategoryID:  "507f1f77bcf86cd799439011",
			},
			wantErr:     true,
			expectedErr: ErrTitleRequired,
		},
		{
			name: "title too long",
			req: models.CreateProductRequest{
				Title:       string(make([]byte, 201)), // 201 characters
				Description: "Description",
				Price:       100,
				Currency:    "USD",
				Images:      []string{"https://example.com/image1.jpg"},
				Location:    "New York, NY",
				CategoryID:  "507f1f77bcf86cd799439011",
			},
			wantErr:     true,
			expectedErr: ErrTitleTooLong,
		},
		{
			name: "invalid price",
			req: models.CreateProductRequest{
				Title:       "Valid Title",
				Description: "Description",
				Price:       -10,
				Currency:    "USD",
				Images:      []string{"https://example.com/image1.jpg"},
				Location:    "New York, NY",
				CategoryID:  "507f1f77bcf86cd799439011",
			},
			wantErr:     true,
			expectedErr: ErrPriceInvalid,
		},
		{
			name: "no images",
			req: models.CreateProductRequest{
				Title:       "Valid Title",
				Description: "Description",
				Price:       100,
				Currency:    "USD",
				Images:      []string{},
				Location:    "New York, NY",
				CategoryID:  "507f1f77bcf86cd799439011",
			},
			wantErr:     true,
			expectedErr: ErrImagesRequired,
		},
		{
			name: "too many tags",
			req: models.CreateProductRequest{
				Title:       "Valid Title",
				Description: "Description",
				Price:       100,
				Currency:    "USD",
				Images:      []string{"https://example.com/image1.jpg"},
				Location:    "New York, NY",
				CategoryID:  "507f1f77bcf86cd799439011",
				Tags:        []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}, // 11 tags
			},
			wantErr:     true,
			expectedErr: ErrInvalidTags,
		},
		{
			name: "empty currency",
			req: models.CreateProductRequest{
				Title:       "Valid Title",
				Description: "Description",
				Price:       100,
				Currency:    "",
				Images:      []string{"https://example.com/image1.jpg"},
				Location:    "New York, NY",
				CategoryID:  "507f1f77bcf86cd799439011",
			},
			wantErr:     true,
			expectedErr: ErrCurrencyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateProductRequest(&tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}