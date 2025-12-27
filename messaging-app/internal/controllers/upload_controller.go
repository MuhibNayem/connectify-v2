package controllers

import (
	"io"
	"messaging-app/internal/storageclient"
	"net/http"

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
