package http

import (
	"errors"
	"net/http"
	"user-service/internal/validation"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserHandler struct {
	userService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile returns the authenticated user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "Authentication required", ErrCodeUnauthorized)
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "User not found", ErrCodeUserNotFound)
		return
	}

	// Clear sensitive data
	user.Password = ""
	RespondWithData(c, http.StatusOK, user)
}

// GetUserByID returns a user by their ID (public profile)
func (h *UserHandler) GetUserByID(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid user ID format", ErrCodeValidation)
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "User not found", ErrCodeUserNotFound)
		return
	}

	// Return public profile only
	RespondWithData(c, http.StatusOK, gin.H{
		"id":        user.ID,
		"username":  user.Username,
		"full_name": user.FullName,
		"avatar":    user.Avatar,
		"bio":       user.Bio,
	})
}

// UpdateProfile updates the authenticated user's profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "Authentication required", ErrCodeUnauthorized)
		return
	}

	var req struct {
		FullName   string `json:"full_name"`
		Bio        string `json:"bio"`
		Avatar     string `json:"avatar"`
		CoverPhoto string `json:"cover_photo"`
		Location   string `json:"location"`
		Website    string `json:"website"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	updatedUser, err := h.userService.UpdateProfileFields(c.Request.Context(), userID, req.FullName, req.Bio, req.Avatar, req.CoverPhoto, req.Location, req.Website)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	updatedUser.Password = ""
	RespondWithSuccess(c, http.StatusOK, "profile updated successfully", updatedUser)
}

// UpdateEmail updates the authenticated user's email
func (h *UserHandler) UpdateEmail(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "invalid user ID", ErrCodeUnauthorized)
		return
	}

	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	// Validate email format
	if err := validation.ValidateEmail(req.Email); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	if err := h.userService.UpdateEmail(c.Request.Context(), userID, req.Email); err != nil {
		if err.Error() == "email already in use by another account" {
			RespondWithError(c, http.StatusConflict, err.Error(), ErrCodeEmailExists)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithSuccess(c, http.StatusOK, "email updated successfully")
}

// UpdatePassword updates the authenticated user's password
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "invalid user ID", ErrCodeUnauthorized)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	// Validate password strength
	if err := validation.ValidatePasswordChange(req.CurrentPassword, req.NewPassword); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	if err := h.userService.UpdatePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if err.Error() == "current password is incorrect" {
			RespondWithError(c, http.StatusUnauthorized, err.Error(), ErrCodeInvalidCredentials)
			return
		}
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithSuccess(c, http.StatusOK, "password updated successfully")
}

// UpdatePrivacySettings updates the authenticated user's privacy settings
func (h *UserHandler) UpdatePrivacySettings(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "invalid user ID", ErrCodeUnauthorized)
		return
	}

	var req models.UpdatePrivacySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	if err := h.userService.UpdatePrivacySettings(c.Request.Context(), userID, &req); err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithSuccess(c, http.StatusOK, "privacy settings updated")
}

// UpdateNotificationSettings updates the authenticated user's notification settings
func (h *UserHandler) UpdateNotificationSettings(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "invalid user ID", ErrCodeUnauthorized)
		return
	}

	var req models.UpdateNotificationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	if err := h.userService.UpdateNotificationSettings(c.Request.Context(), userID, &req); err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithSuccess(c, http.StatusOK, "notification settings updated")
}

// ToggleTwoFactor enables or disables two-factor authentication
func (h *UserHandler) ToggleTwoFactor(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "invalid user ID", ErrCodeUnauthorized)
		return
	}

	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		return
	}

	if err := h.userService.ToggleTwoFactor(c.Request.Context(), userID, req.Enable); err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	status := "disabled"
	if req.Enable {
		status = "enabled"
	}
	RespondWithSuccess(c, http.StatusOK, "two-factor authentication "+status)
}

// DeactivateAccount deactivates the authenticated user's account
func (h *UserHandler) DeactivateAccount(c *gin.Context) {
	userID, err := h.extractUserID(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "invalid user ID", ErrCodeUnauthorized)
		return
	}

	if err := h.userService.DeactivateAccount(c.Request.Context(), userID); err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithSuccess(c, http.StatusOK, "account deactivated")
}

// GetUserStatus returns the online/offline status of a user
func (h *UserHandler) GetUserStatus(c *gin.Context) {
	idParam := c.Param("id")

	status, lastSeen, err := h.userService.GetUserStatus(c.Request.Context(), idParam)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error(), ErrCodeInternalError)
		return
	}

	RespondWithData(c, http.StatusOK, gin.H{
		"status":    status,
		"last_seen": lastSeen,
	})
}

// extractUserID extracts user ID from JWT claims in context
func (h *UserHandler) extractUserID(c *gin.Context) (primitive.ObjectID, error) {
	// User ID is set by auth middleware
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return primitive.NilObjectID, errors.New("authentication required")
	}

	switch v := userIDStr.(type) {
	case string:
		return primitive.ObjectIDFromHex(v)
	case primitive.ObjectID:
		return v, nil
	default:
		return primitive.NilObjectID, nil
	}
}
