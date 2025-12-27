package httpapi

import (
	"io"
	"net/http"
	"time"

	"github.com/MuhibNayem/connectify-v2/storage-service/internal/service"
	"github.com/gin-gonic/gin"
)

type StorageHandler struct {
	svc *service.StorageService
}

func NewStorageHandler(svc *service.StorageService) *StorageHandler {
	return &StorageHandler{svc: svc}
}

func (h *StorageHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/storage")
	{
		api.POST("/upload", h.Upload)
		api.POST("/upload-multiple", h.UploadMultiple)
		api.DELETE("/delete/:key", h.Delete)
		api.DELETE("/delete-by-url", h.DeleteByURL)
		api.POST("/archive", h.UploadArchive)
		api.GET("/archive/:path", h.DownloadArchive)
		api.GET("/presigned/:key", h.GetPresignedURL)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

func (h *StorageHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	result, err := h.svc.Upload(c.Request.Context(), file, header.Size, header.Filename, header.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *StorageHandler) UploadMultiple(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files"})
		return
	}

	uploads := make([]service.FileUpload, 0, len(files))
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file"})
			return
		}
		defer f.Close()

		data, _ := io.ReadAll(f)
		uploads = append(uploads, service.FileUpload{
			Data:        io.NopCloser(io.NewSectionReader(f, 0, fh.Size)),
			Size:        fh.Size,
			Filename:    fh.Filename,
			ContentType: fh.Header.Get("Content-Type"),
		})
		_ = data
	}

	results, err := h.svc.UploadMultiple(c.Request.Context(), uploads)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *StorageHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key required"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *StorageHandler) DeleteByURL(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.DeleteByURL(c.Request.Context(), req.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *StorageHandler) UploadArchive(c *gin.Context) {
	var req struct {
		ObjectPath string `json:"object_path" binding:"required"`
		Data       []byte `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.UploadArchive(c.Request.Context(), req.ObjectPath, req.Data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *StorageHandler) DownloadArchive(c *gin.Context) {
	path := c.Param("path")
	data, err := h.svc.DownloadArchive(c.Request.Context(), path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

func (h *StorageHandler) GetPresignedURL(c *gin.Context) {
	key := c.Param("key")
	expiryStr := c.DefaultQuery("expiry", "900")
	var expiry time.Duration
	if seconds, err := time.ParseDuration(expiryStr + "s"); err == nil {
		expiry = seconds
	} else {
		expiry = 15 * time.Minute
	}

	url, err := h.svc.GetPresignedURL(c.Request.Context(), key, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
