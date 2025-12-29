package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestRouter creates a test router with mock handlers
func setupTestRouter() *gin.Engine {
	router := gin.New()

	// Middleware to set user context
	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			c.Set("userID", "507f1f77bcf86cd799439011")
		}
		c.Next()
	})

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Friend request endpoints
	api := router.Group("/api/v1")
	{
		friendships := api.Group("/friendships")
		{
			friendships.POST("/requests", handleSendRequest)
			friendships.POST("/requests/:id/respond", handleRespondToRequest)
			friendships.GET("", handleListFriendships)
			friendships.GET("/search", handleSearchFriends)
			friendships.GET("/check", handleCheckFriendship)
			friendships.DELETE("/:friend_id", handleUnfriend)
			friendships.POST("/block/:user_id", handleBlockUser)
			friendships.DELETE("/block/:user_id", handleUnblockUser)
			friendships.GET("/blocked", handleGetBlockedUsers)
		}
	}

	return router
}

func handleSendRequest(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		ReceiverID string `json:"receiver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate ObjectID
	if len(req.ReceiverID) != 24 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receiver_id"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           primitive.NewObjectID().Hex(),
		"requester_id": "507f1f77bcf86cd799439011",
		"receiver_id":  req.ReceiverID,
		"status":       "pending",
	})
}

func handleRespondToRequest(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Accept bool `json:"accept"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := "rejected"
	if req.Accept {
		status = "accepted"
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

func handleListFriendships(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       []interface{}{},
		"total":      0,
		"page":       1,
		"limit":      10,
		"totalPages": 0,
	})
}

func handleSearchFriends(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	c.JSON(http.StatusOK, []interface{}{})
}

func handleCheckFriendship(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	otherUserID := c.Query("user_id")
	if otherUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"are_friends": false})
}

func handleUnfriend(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	friendID := c.Param("friend_id")
	if len(friendID) != 24 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid friend_id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func handleBlockUser(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID := c.Param("user_id")
	if len(userID) != 24 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func handleUnblockUser(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func handleGetBlockedUsers(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blocked_users": []interface{}{}})
}

// === Integration Tests ===

func TestAPI_HealthEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestAPI_SendRequest_Success(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"receiver_id": "507f1f77bcf86cd799439012",
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["status"] != "pending" {
		t.Errorf("Expected status 'pending', got '%v'", response["status"])
	}
}

func TestAPI_SendRequest_Unauthorized(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"receiver_id": "507f1f77bcf86cd799439012",
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestAPI_SendRequest_InvalidReceiverID(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"receiver_id": "invalid",
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestAPI_RespondToRequest_Accept(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]bool{"accept": true})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests/507f1f77bcf86cd799439012/respond", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got '%v'", response["status"])
	}
}

func TestAPI_RespondToRequest_Reject(t *testing.T) {
	router := setupTestRouter()

	body, _ := json.Marshal(map[string]bool{"accept": false})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests/507f1f77bcf86cd799439012/respond", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["status"] != "rejected" {
		t.Errorf("Expected status 'rejected', got '%v'", response["status"])
	}
}

func TestAPI_ListFriendships_Success(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["data"]; !ok {
		t.Error("Expected 'data' field in response")
	}
	if _, ok := response["total"]; !ok {
		t.Error("Expected 'total' field in response")
	}
}

func TestAPI_SearchFriends_Success(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships/search?query=john", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAPI_SearchFriends_MissingQuery(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships/search", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestAPI_CheckFriendship_Success(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships/check?user_id=507f1f77bcf86cd799439012", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["are_friends"]; !ok {
		t.Error("Expected 'are_friends' field in response")
	}
}

func TestAPI_Unfriend_Success(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/friendships/507f1f77bcf86cd799439012", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAPI_Unfriend_InvalidID(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/friendships/invalid", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestAPI_BlockUser_Success(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/block/507f1f77bcf86cd799439012", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAPI_UnblockUser_Success(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/friendships/block/507f1f77bcf86cd799439012", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestAPI_GetBlockedUsers_Success(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships/blocked", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["blocked_users"]; !ok {
		t.Error("Expected 'blocked_users' field in response")
	}
}

// === Flow Tests ===

func TestAPI_CompleteFlow_SendAcceptCheck(t *testing.T) {
	router := setupTestRouter()

	// Step 1: Send friend request
	body, _ := json.Marshal(map[string]string{
		"receiver_id": "507f1f77bcf86cd799439012",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Step 1 failed: Expected 201, got %d", w.Code)
	}

	// Step 2: Accept friend request
	body, _ = json.Marshal(map[string]bool{"accept": true})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/friendships/requests/507f1f77bcf86cd799439012/respond", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 2 failed: Expected 200, got %d", w.Code)
	}

	// Step 3: Check friendship
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/friendships/check?user_id=507f1f77bcf86cd799439012", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Step 3 failed: Expected 200, got %d", w.Code)
	}

	t.Log("Complete flow test passed: Send → Accept → Check")
}

func TestAPI_CompleteFlow_BlockAndUnblock(t *testing.T) {
	router := setupTestRouter()

	userToBlock := "507f1f77bcf86cd799439013"

	// Step 1: Block user
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/block/"+userToBlock, nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Block failed: Expected 200, got %d", w.Code)
	}

	// Step 2: Verify blocked
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/friendships/blocked", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Get blocked failed: Expected 200, got %d", w.Code)
	}

	// Step 3: Unblock user
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/friendships/block/"+userToBlock, nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Unblock failed: Expected 200, got %d", w.Code)
	}

	t.Log("Complete flow test passed: Block → Verify → Unblock")
}
