package controllers

import (
	"net/http"
	"strings"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MarketplaceController struct {
	service MarketplaceService
}

func NewMarketplaceController(svc MarketplaceService) *MarketplaceController {
	return &MarketplaceController{service: svc}
}

func (c *MarketplaceController) GetCategories(ctx *gin.Context) {
	categories, err := c.service.GetCategories(ctx.Request.Context())
	if err != nil {
		RespondWithError(ctx, http.StatusInternalServerError, "Failed to fetch categories", ErrCodeInternalError)
		return
	}
	RespondWithData(ctx, http.StatusOK, categories)
}

func (c *MarketplaceController) CreateProduct(ctx *gin.Context) {
	userIDStr, ok := ExtractUserID(ctx)
	if !ok {
		RespondWithError(ctx, http.StatusUnauthorized, "Authentication required", ErrCodeUnauthorized)
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		RespondWithError(ctx, http.StatusUnauthorized, "Invalid user authentication", ErrCodeUnauthorized)
		return
	}

	var req models.CreateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		RespondWithError(ctx, http.StatusBadRequest, "Invalid request format", ErrCodeValidation)
		return
	}

	if len(req.Images) > 5 {
		RespondWithValidationError(ctx, "Too many images", map[string]string{
			"images": "Maximum 5 images allowed per product",
		})
		return
	}

	product, err := c.service.CreateProduct(ctx.Request.Context(), userObjectID, req)
	if err != nil {
		if strings.Contains(err.Error(), "validation") {
			RespondWithError(ctx, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		} else {
			RespondWithError(ctx, http.StatusInternalServerError, "Failed to create product", ErrCodeInternalError)
		}
		return
	}
	RespondWithSuccess(ctx, http.StatusCreated, "Product created successfully", product)
}

func (c *MarketplaceController) GetProduct(ctx *gin.Context) {
	productIDStr := ctx.Param("id")
	productID, err := primitive.ObjectIDFromHex(productIDStr)
	if err != nil {
		RespondWithError(ctx, http.StatusBadRequest, "Invalid product ID format", ErrCodeInvalidProductID)
		return
	}

	// Get viewer ID for personalization (saved status)
	viewerID := primitive.NilObjectID
	if userIDStr, ok := ExtractUserID(ctx); ok {
		if vid, err := primitive.ObjectIDFromHex(userIDStr); err == nil {
			viewerID = vid
		}
	}

	product, err := c.service.GetProductByID(ctx.Request.Context(), productID, viewerID)
	if err != nil {
		RespondWithError(ctx, http.StatusNotFound, "Product not found", ErrCodeProductNotFound)
		return
	}
	RespondWithData(ctx, http.StatusOK, product)
}

func (c *MarketplaceController) SearchProducts(ctx *gin.Context) {
	var filter models.ProductFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		RespondWithError(ctx, http.StatusBadRequest, "Invalid search parameters", ErrCodeValidation)
		return
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}

	resp, err := c.service.SearchProducts(ctx.Request.Context(), filter)
	if err != nil {
		RespondWithError(ctx, http.StatusInternalServerError, "Search failed", ErrCodeInternalError)
		return
	}

	RespondWithData(ctx, http.StatusOK, resp)
}

func (c *MarketplaceController) GetMarketplaceConversations(ctx *gin.Context) {
	userIDStr, ok := ExtractUserID(ctx)
	if !ok {
		RespondWithError(ctx, http.StatusUnauthorized, "Authentication required", ErrCodeUnauthorized)
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		RespondWithError(ctx, http.StatusUnauthorized, "Invalid user authentication", ErrCodeUnauthorized)
		return
	}

	conversations, err := c.service.GetMarketplaceConversations(ctx.Request.Context(), userObjectID)
	if err != nil {
		RespondWithError(ctx, http.StatusInternalServerError, "Failed to fetch conversations", ErrCodeInternalError)
		return
	}

	if conversations == nil {
		conversations = []models.ConversationSummary{}
	}

	RespondWithData(ctx, http.StatusOK, conversations)
}

func (c *MarketplaceController) MarkProductSold(ctx *gin.Context) {
	productIDStr := ctx.Param("id")
	productID, err := primitive.ObjectIDFromHex(productIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	userID, _ := ctx.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}
	userObjectID, _ := primitive.ObjectIDFromHex(userIDStr)

	if err := c.service.MarkProductSold(ctx.Request.Context(), productID, userObjectID); err != nil {
		if err.Error() == "unauthorized" {
			RespondWithError(ctx, http.StatusForbidden, "You can only mark your own products as sold", ErrCodeInsufficientPerms)
			return
		}
		RespondWithError(ctx, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithSuccess(ctx, http.StatusOK, "Product marked as sold")
}

func (c *MarketplaceController) DeleteProduct(ctx *gin.Context) {
	productIDStr := ctx.Param("id")
	productID, err := primitive.ObjectIDFromHex(productIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	userID, _ := ctx.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}
	userObjectID, _ := primitive.ObjectIDFromHex(userIDStr)

	if err := c.service.DeleteProduct(ctx.Request.Context(), productID, userObjectID); err != nil {
		if err.Error() == "unauthorized" {
			RespondWithError(ctx, http.StatusForbidden, "You can only delete your own products", ErrCodeInsufficientPerms)
			return
		}
		RespondWithError(ctx, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithSuccess(ctx, http.StatusOK, "Product deleted successfully")
}

func (c *MarketplaceController) ToggleSaveProduct(ctx *gin.Context) {
	productIDStr := ctx.Param("id")
	productID, err := primitive.ObjectIDFromHex(productIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	userID, _ := ctx.Get("userID")
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}
	userObjectID, _ := primitive.ObjectIDFromHex(userIDStr)

	isSaved, err := c.service.ToggleSaveProduct(ctx.Request.Context(), productID, userObjectID)
	if err != nil {
		RespondWithError(ctx, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	message := "Product saved successfully"
	if !isSaved {
		message = "Product removed from saved items"
	}
	RespondWithSuccess(ctx, http.StatusOK, message, gin.H{"saved": isSaved})
}

func (c *MarketplaceController) UpdateProduct(ctx *gin.Context) {
	ctx.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
