package marketplace

import (
	"time"

	"gitlab.com/spydotech-group/shared-entity/models"
	marketplacepb "gitlab.com/spydotech-group/shared-entity/proto/marketplace/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProtoProductToModel converts a proto Product to models.Product
func ProtoProductToModel(p *marketplacepb.Product) *models.Product {
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
		Location:    p.Location.City, // Proto has struct, model has string
		Status:      models.ProductStatus(p.Status),
		Tags:        p.Tags,
		Views:       p.Views,
		CreatedAt:   p.CreatedAt.AsTime(),
		UpdatedAt:   time.Now(),
	}
}

// ProtoProductToResponse converts a proto Product to models.ProductResponse
func ProtoProductToResponse(p *marketplacepb.Product) *models.ProductResponse {
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
		Location:    p.Location.City, // Proto has struct, model has string
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

// ToProtoProductFromModel converts models.Product to proto Product
func ToProtoProductFromModel(product *models.Product) *marketplacepb.Product {
	if product == nil {
		return nil
	}

	// Handle location conversion - model is string, proto is struct
	location := &marketplacepb.Location{}
	if product.Location != "" {
		location.City = product.Location
	}

	return &marketplacepb.Product{
		Id:          product.ID.Hex(),
		Title:       product.Title,
		Description: product.Description,
		Price:       product.Price,
		Currency:    product.Currency,
		Images:      product.Images,
		Location:    location,
		Status:      string(product.Status),
		Tags:        product.Tags,
		Views:       product.Views,
		CreatedAt:   timestamppb.New(product.CreatedAt),
		Seller: &marketplacepb.UserShort{
			// Note: Product model only has SellerID, not full Seller info.
			// This might be incomplete if caller expects full seller info.
			Id: product.SellerID.Hex(),
		},
		Category: &marketplacepb.Category{
			// Note: Product model only has CategoryID
			Id: product.CategoryID.Hex(),
		},
		// IsSaved is not in Product model
	}
}

// ToProtoProduct converts models.ProductResponse to proto Product
func ToProtoProduct(product *models.ProductResponse) *marketplacepb.Product {
	if product == nil {
		return nil
	}

	// Handle location conversion - model is string, proto is struct
	location := &marketplacepb.Location{}
	if product.Location != "" {
		location.City = product.Location
	}

	return &marketplacepb.Product{
		Id:          product.ID.Hex(),
		Title:       product.Title,
		Description: product.Description,
		Price:       product.Price,
		Currency:    product.Currency,
		Images:      product.Images,
		Location:    location,
		Status:      string(product.Status),
		Tags:        product.Tags,
		Views:       product.Views,
		CreatedAt:   timestamppb.New(product.CreatedAt),
		Seller: &marketplacepb.UserShort{
			Id:       product.Seller.ID.Hex(),
			Username: product.Seller.Username,
			FullName: product.Seller.FullName,
			Avatar:   product.Seller.Avatar,
		},
		Category: &marketplacepb.Category{
			Id:   product.Category.ID.Hex(),
			Name: product.Category.Name,
			Slug: product.Category.Slug,
			Icon: product.Category.Icon,
		},
		IsSaved: product.IsSaved,
	}
}

// ToProtoProducts converts a slice of ProductResponse to proto Products
func ToProtoProducts(products []models.ProductResponse) []*marketplacepb.Product {
	if len(products) == 0 {
		return nil
	}

	result := make([]*marketplacepb.Product, 0, len(products))
	for i := range products {
		if p := ToProtoProduct(&products[i]); p != nil {
			result = append(result, p)
		}
	}
	return result
}

// ToProtoCategories converts models.Category slice to proto Categories
func ToProtoCategories(categories []models.Category) []*marketplacepb.Category {
	if len(categories) == 0 {
		return nil
	}

	result := make([]*marketplacepb.Category, 0, len(categories))
	for _, cat := range categories {
		result = append(result, &marketplacepb.Category{
			Id:    cat.ID.Hex(),
			Name:  cat.Name,
			Slug:  cat.Slug,
			Icon:  cat.Icon,
			Order: int32(cat.Order),
		})
	}
	return result
}

// ToProtoConversations converts models.ConversationSummary slice to proto ConversationSummaries
func ToProtoConversations(conversations []models.ConversationSummary) []*marketplacepb.ConversationSummary {
	if len(conversations) == 0 {
		return nil
	}

	result := make([]*marketplacepb.ConversationSummary, 0, len(conversations))
	for _, conv := range conversations {
		var timestamp *timestamppb.Timestamp
		if conv.LastMessageTimestamp != nil {
			timestamp = timestamppb.New(*conv.LastMessageTimestamp)
		}

		result = append(result, &marketplacepb.ConversationSummary{
			Id:                     conv.ID,
			Name:                   conv.Name,
			Avatar:                 conv.Avatar,
			IsGroup:                conv.IsGroup,
			LastMessageContent:     conv.LastMessageContent,
			LastMessageTimestamp:   timestamp,
			LastMessageSenderId:    conv.LastMessageSenderID.Hex(),
			LastMessageSenderName:  conv.LastMessageSenderName,
			UnreadCount:            int32(conv.UnreadCount),
			LastMessageIsEncrypted: conv.LastMessageIsEncrypted,
		})
	}
	return result
}

// ToTimestamp converts time.Time to proto Timestamp
func ToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// ObjectIDFromString converts string to primitive.ObjectID
func ObjectIDFromString(value string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(value)
}
