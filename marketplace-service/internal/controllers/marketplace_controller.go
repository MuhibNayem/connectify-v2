package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MarketplaceController struct {
	service *service.MarketplaceService
}

func NewMarketplaceController(svc *service.MarketplaceService) *MarketplaceController {
	return &MarketplaceController{service: svc}
}

func (c *MarketplaceController) GetCategories(ctx *gin.Context) {
	categories, err := c.service.GetCategories(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, categories)
}

func (c *MarketplaceController) CreateProduct(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	var req models.CreateProductRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Images) > 5 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "maximum 5 images allowed"})
		return
	}

	product, err := c.service.CreateProduct(ctx.Request.Context(), userObjectID, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, product)
}

func (c *MarketplaceController) GetProduct(ctx *gin.Context) {
	productIDStr := ctx.Param("id")
	productID, err := primitive.ObjectIDFromHex(productIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var viewerID primitive.ObjectID
	if vID, exists := ctx.Get("userID"); exists {
		if vIDStr, ok := vID.(string); ok {
			viewerID, _ = primitive.ObjectIDFromHex(vIDStr)
		}
	}

	product, err := c.service.GetProductByID(ctx.Request.Context(), productID, viewerID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	ctx.JSON(http.StatusOK, product)
}

func (c *MarketplaceController) SearchProducts(ctx *gin.Context) {
	var filter models.ProductFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (c *MarketplaceController) GetMarketplaceConversations(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in context"})
		return
	}
	userObjectID, _ := primitive.ObjectIDFromHex(userIDStr)

	conversations, err := c.service.GetMarketplaceConversations(ctx.Request.Context(), userObjectID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if conversations == nil {
		conversations = []models.ConversationSummary{}
	}

	ctx.JSON(http.StatusOK, conversations)
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
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
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"saved": isSaved})
}

func (c *MarketplaceController) UpdateProduct(ctx *gin.Context) {
	ctx.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
