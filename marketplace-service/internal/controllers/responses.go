package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard error codes for marketplace service
const (
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeProductNotFound   = "PRODUCT_NOT_FOUND"
	ErrCodeCategoryNotFound  = "CATEGORY_NOT_FOUND"
	ErrCodeInvalidProductID  = "INVALID_PRODUCT_ID"
	ErrCodeTooManyImages     = "TOO_MANY_IMAGES"
	ErrCodeInsufficientPerms = "INSUFFICIENT_PERMISSIONS"
	ErrCodeRateLimited       = "RATE_LIMITED"
	ErrCodeInternalError     = "INTERNAL_ERROR"
)

// RespondWithError sends a standardized error response
func RespondWithError(c *gin.Context, statusCode int, message string, code ...string) {
	errorCode := ErrCodeInternalError
	if len(code) > 0 {
		errorCode = code[0]
	}
	
	c.JSON(statusCode, ErrorResponse{
		Error: message,
		Code:  errorCode,
	})
}

// RespondWithValidationError sends a validation error with details
func RespondWithValidationError(c *gin.Context, message string, details map[string]string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   message,
		Code:    ErrCodeValidation,
		Details: details,
	})
}

// RespondWithSuccess sends a standardized success response
func RespondWithSuccess(c *gin.Context, statusCode int, message string, data ...interface{}) {
	response := SuccessResponse{
		Message: message,
	}
	
	if len(data) > 0 {
		response.Data = data[0]
	}
	
	c.JSON(statusCode, response)
}

// RespondWithData sends data without a message wrapper
func RespondWithData(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// Helper functions for common patterns
func ExtractUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return "", false
	}
	
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}