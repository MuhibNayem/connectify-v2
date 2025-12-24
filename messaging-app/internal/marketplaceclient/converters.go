package marketplaceclient

import (
	"time"

	"gitlab.com/spydotech-group/shared-entity/models"
	marketplacepb "gitlab.com/spydotech-group/shared-entity/proto/marketplace/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// protoProductToModel converts proto Product to models.Product
func protoProductToModel(p *marketplacepb.Product) *models.Product {
	if p == nil {
		return nil
	}

	id, _ := primitive.ObjectIDFromHex(p.Id)
	sellerID, _ := primitive.ObjectIDFromHex(p.Seller.Id)
	categoryID, _ := primitive.ObjectIDFromHex(p.Category.Id)

	return &models.Product{
		ID:          id,
		SellerID:    sellerID,
		CategoryID:  categoryID,
		Title:       p.Title,
		Description: p.Description,
		Price:       p.Price,
		Currency:    p.Currency,
		Images:      p.Images,
		Location:    protoLocationToModel(p.Location),
		Status:      models.ProductStatus(p.Status),
		Tags:        p.Tags,
		Views:       p.Views,
		CreatedAt:   p.CreatedAt.AsTime(),
		UpdatedAt:   time.Now(),
	}
}

// protoProductToResponse converts proto Product to models.ProductResponse
func protoProductToResponse(p *marketplacepb.Product) *models.ProductResponse {
	if p == nil {
		return nil
	}

	id, _ := primitive.ObjectIDFromHex(p.Id)
	sellerID, _ := primitive.ObjectIDFromHex(p.Seller.Id)
	categoryID, _ := primitive.ObjectIDFromHex(p.Category.Id)

	return &models.ProductResponse{
		ID:          id,
		Title:       p.Title,
		Description: p.Description,
		Price:       p.Price,
		Currency:    p.Currency,
		Images:      p.Images,
		Location:    protoLocationToModel(p.Location),
		Status:      models.ProductStatus(p.Status),
		Tags:        p.Tags,
		Views:       p.Views,
		CreatedAt:   p.CreatedAt.AsTime(),
		Seller: models.UserShortResponse{
			ID:       sellerID,
			Username: p.Seller.Username,
			FullName: p.Seller.FullName,
			Avatar:   p.Seller.Avatar,
		},
		Category: models.Category{
			ID:   categoryID,
			Name: p.Category.Name,
			Slug: p.Category.Slug,
			Icon: p.Category.Icon,
		},
		IsSaved: p.IsSaved,
	}
}

// protoLocationToModel converts proto Location to models.ProductLocation
func protoLocationToModel(loc *marketplacepb.Location) models.ProductLocation {
	if loc == nil {
		return models.ProductLocation{}
	}
	return models.ProductLocation{
		City:      loc.City,
		State:     loc.State,
		Country:   loc.Country,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
	}
}
