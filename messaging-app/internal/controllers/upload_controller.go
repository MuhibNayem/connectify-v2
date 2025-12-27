package controllers

import (
	"io"
	"messaging-app/internal/storageclient"
	"net/http"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/gin-gonic/gin"
)

type UploadController struct {
	storageClient *storageclient.Client
}

func NewUploadController(storageClient *storageclient.Client) *UploadController {
	return &UploadController{storageClient: storageClient}
}

// Upload godoc
// @Summary Upload files
// @Security BearerAuth
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Files to upload"
// @Success 200 {object} []models.MediaItem
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/upload [post]
func (c *UploadController) Upload(ctx *gin.Context) {
	if c.storageClient == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not available"})
		return
	}

	// Parse multipart form
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form: " + err.Error()})
		return
	}

	files := form.File["files[]"]
	if len(files) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	// Enforce file size limit (e.g., 100MB per file)
	const MaxFileSize = 100 * 1024 * 1024 // 100MB
	for _, file := range files {
		if file.Size > MaxFileSize {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 100MB)"})
			return
		}
	}

	var mediaItems []models.MediaItem
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file: " + err.Error()})
			return
		}
		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file: " + err.Error()})
			return
		}

		result, err := c.storageClient.Upload(ctx.Request.Context(), data, fileHeader.Filename, fileHeader.Header.Get("Content-Type"))
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file: " + err.Error()})
			return
		}

		mediaItems = append(mediaItems, models.MediaItem{
			URL:  result.URL,
			Type: result.Type,
		})
	}

	ctx.JSON(http.StatusOK, mediaItems)
}

// GetPresignedDownloadURL godoc
// @Summary Get a presigned URL for downloading a file
// @Security BearerAuth
// @Tags storage
// @Produce json
// @Param key query string true "Storage key of the file"
// @Success 200 {object} gin.H{"url": "string"}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/storage/download-url [get]
func (c *UploadController) GetPresignedDownloadURL(ctx *gin.Context) {
	if c.storageClient == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not available"})
		return
	}

	key := ctx.Query("key")
	if key == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	// Generate presigned URL valid for 15 minutes
	url, err := c.storageClient.GetPresignedURL(ctx.Request.Context(), key, 15*time.Minute)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate presigned URL: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"url":        url,
		"expires_in": 900, // 15 minutes in seconds
	})
}

// GetPresignedUploadURL godoc
// @Summary Get a presigned URL for direct-to-S3 upload (FAANG scale)
// @Security BearerAuth
// @Tags storage
// @Accept json
// @Produce json
// @Param request body object{filename=string,content_type=string,content_length=int64,sha256_hash=string} true "Upload request"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/storage/upload-url [post]
func (c *UploadController) GetPresignedUploadURL(ctx *gin.Context) {
	if c.storageClient == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage service not available"})
		return
	}

	var req struct {
		Filename      string `json:"filename" binding:"required"`
		ContentType   string `json:"content_type" binding:"required"`
		ContentLength int64  `json:"content_length" binding:"required"`
		Sha256Hash    string `json:"sha256_hash" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := c.storageClient.GetPresignedUploadURL(ctx.Request.Context(), req.Filename, req.ContentType, req.Sha256Hash, req.ContentLength)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate presigned upload URL: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"upload_url":   result.UploadURL,
		"file_url":     result.FileURL,
		"key":          result.Key,
		"is_duplicate": result.IsDuplicate,
	})
}
