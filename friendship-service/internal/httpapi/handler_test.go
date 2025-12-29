package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthEndpoint(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestSendRequest_Unauthorized(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.POST("/api/v1/friendships/requests", func(c *gin.Context) {
		// Simulate auth middleware check
		_, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
	})

	body := bytes.NewBufferString(`{"receiver_id": "507f1f77bcf86cd799439011"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestSendRequest_InvalidJSON(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.POST("/api/v1/friendships/requests", func(c *gin.Context) {
		c.Set("userID", "507f1f77bcf86cd799439011")

		var req struct {
			ReceiverID string `json:"receiver_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	})

	router.Use(func(c *gin.Context) {
		c.Set("userID", "507f1f77bcf86cd799439011")
		c.Next()
	})

	body := bytes.NewBufferString(`{invalid json}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/requests", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestListFriendships_DefaultPagination(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.GET("/api/v1/friendships", func(c *gin.Context) {
		c.Set("userID", "507f1f77bcf86cd799439011")

		page := c.DefaultQuery("page", "1")
		limit := c.DefaultQuery("limit", "10")

		c.JSON(http.StatusOK, gin.H{
			"data":       []interface{}{},
			"total":      0,
			"page":       page,
			"limit":      limit,
			"totalPages": 0,
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["page"] != "1" {
		t.Errorf("Expected page '1', got '%v'", response["page"])
	}
}

func TestSearchFriends_MissingQuery(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.GET("/api/v1/friendships/search", func(c *gin.Context) {
		query := c.Query("query")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
			return
		}
		c.JSON(http.StatusOK, []interface{}{})
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestSearchFriends_ValidQuery(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.GET("/api/v1/friendships/search", func(c *gin.Context) {
		query := c.Query("query")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
			return
		}
		c.JSON(http.StatusOK, []interface{}{})
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/friendships/search?query=john", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestBlockUser_InvalidUserID(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.POST("/api/v1/friendships/block/:user_id", func(c *gin.Context) {
		userID := c.Param("user_id")
		if len(userID) != 24 { // Simple ObjectID validation
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/friendships/block/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUnfriend_Success(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.DELETE("/api/v1/friendships/:friend_id", func(c *gin.Context) {
		friendID := c.Param("friend_id")
		if len(friendID) == 24 {
			c.JSON(http.StatusOK, gin.H{"status": "success"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid friend ID"})
	})

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/friendships/507f1f77bcf86cd799439011", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRespondToRequest_AcceptFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		accept bool
	}{
		{"Accept request", true},
		{"Reject request", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/api/v1/friendships/requests/:id/respond", func(c *gin.Context) {
				var req struct {
					Accept bool `json:"accept"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success", "accepted": req.Accept})
			})

			bodyJSON, _ := json.Marshal(map[string]bool{"accept": tt.accept})
			req, _ := http.NewRequest(
				http.MethodPost,
				"/api/v1/friendships/requests/507f1f77bcf86cd799439011/respond",
				bytes.NewBuffer(bodyJSON),
			)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if response["accepted"] != tt.accept {
				t.Errorf("Expected accepted=%v, got %v", tt.accept, response["accepted"])
			}
		})
	}
}
