package grpc

import (
	"context"
	"log/slog"

	marketplace "gitlab.com/spydotech-group/marketplace-service/internal"
	"gitlab.com/spydotech-group/marketplace-service/internal/service"
	"gitlab.com/spydotech-group/shared-entity/models"
	marketplacepb "gitlab.com/spydotech-group/shared-entity/proto/marketplace/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	marketplacepb.UnimplementedMarketplaceServiceServer
	service *service.MarketplaceService
}

func NewServer(svc *service.MarketplaceService) *Server {
	return &Server{
		service: svc,
	}
}

func (s *Server) CreateProduct(ctx context.Context, req *marketplacepb.CreateProductRequest) (*marketplacepb.ProductResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	// Convert proto Location to string (models use string for location)
	locationStr := req.Location.City
	if req.Location.State != "" {
		locationStr += ", " + req.Location.State
	}
	if req.Location.Country != "" {
		locationStr += ", " + req.Location.Country
	}

	createReq := models.CreateProductRequest{
		CategoryID:  req.CategoryId,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Currency:    req.Currency,
		Images:      req.Images,
		Location:    locationStr, // Product.Location is a string
		Tags:        req.Tags,
	}

	product, err := s.service.CreateProduct(ctx, userID, createReq)
	if err != nil {
		slog.Error("Error creating product", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	return &marketplacepb.ProductResponse{
		Product: marketplace.ToProtoProductFromModel(product),
	}, nil
}

func (s *Server) GetProduct(ctx context.Context, req *marketplacepb.GetProductRequest) (*marketplacepb.ProductResponse, error) {
	productID, err := primitive.ObjectIDFromHex(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
	}

	viewerID, err := primitive.ObjectIDFromHex(req.ViewerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid viewer ID: %v", err)
	}

	productResp, err := s.service.GetProductByID(ctx, productID, viewerID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "product not found: %v", err)
	}

	return &marketplacepb.ProductResponse{
		Product: marketplace.ToProtoProduct(productResp),
	}, nil
}

func (s *Server) SearchProducts(ctx context.Context, req *marketplacepb.SearchProductsRequest) (*marketplacepb.SearchProductsResponse, error) {
	filter := models.ProductFilter{
		CategoryID: req.CategoryId,
		Query:      req.Query,
		MinPrice:   &req.MinPrice,
		MaxPrice:   &req.MaxPrice,
		SortBy:     req.SortBy,
		Page:       req.Page,
		Limit:      req.Limit,
	}

	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	result, err := s.service.SearchProducts(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search products: %v", err)
	}

	products := make([]*marketplacepb.Product, len(result.Products))
	for i, p := range result.Products {
		products[i] = marketplace.ToProtoProduct(&p)
	}

	return &marketplacepb.SearchProductsResponse{
		Products: products,
		Total:    result.Total,
		Page:     result.Page,
		Limit:    result.Limit,
	}, nil
}

func (s *Server) GetCategories(ctx context.Context, _ *emptypb.Empty) (*marketplacepb.GetCategoriesResponse, error) {
	categories, err := s.service.GetCategories(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get categories: %v", err)
	}

	pbCategories := make([]*marketplacepb.Category, len(categories))
	for i, cat := range categories {
		pbCategories[i] = &marketplacepb.Category{
			Id:    cat.ID.Hex(),
			Name:  cat.Name,
			Slug:  cat.Slug,
			Icon:  cat.Icon,
			Order: int32(cat.Order),
		}
	}

	return &marketplacepb.GetCategoriesResponse{
		Categories: pbCategories,
	}, nil
}

func (s *Server) ToggleSaveProduct(ctx context.Context, req *marketplacepb.ToggleSaveProductRequest) (*marketplacepb.ToggleSaveProductResponse, error) {
	productID, err := primitive.ObjectIDFromHex(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
	}

	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	isSaved, err := s.service.ToggleSaveProduct(ctx, productID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to toggle save: %v", err)
	}

	return &marketplacepb.ToggleSaveProductResponse{
		IsSaved: isSaved,
	}, nil
}

func (s *Server) MarkProductSold(ctx context.Context, req *marketplacepb.MarkProductSoldRequest) (*emptypb.Empty, error) {
	productID, err := primitive.ObjectIDFromHex(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
	}

	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	if err := s.service.MarkProductSold(ctx, productID, userID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mark as sold: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) DeleteProduct(ctx context.Context, req *marketplacepb.DeleteProductRequest) (*emptypb.Empty, error) {
	productID, err := primitive.ObjectIDFromHex(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
	}

	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	if err := s.service.DeleteProduct(ctx, productID, userID); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "failed to delete product: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetMarketplaceConversations(ctx context.Context, req *marketplacepb.GetConversationsRequest) (*marketplacepb.GetConversationsResponse, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	conversations, err := s.service.GetMarketplaceConversations(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversations: %v", err)
	}

	return &marketplacepb.GetConversationsResponse{
		Conversations: marketplace.ToProtoConversations(conversations),
	}, nil
}

func (s *Server) UpdateProduct(ctx context.Context, req *marketplacepb.UpdateProductRequest) (*marketplacepb.ProductResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "UpdateProduct not implemented yet")
}

func (s *Server) GetSavedProducts(ctx context.Context, req *marketplacepb.GetSavedProductsRequest) (*marketplacepb.SearchProductsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetSavedProducts not implemented yet")
}
