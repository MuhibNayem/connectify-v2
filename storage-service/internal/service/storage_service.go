package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/storage-service/config"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageService struct {
	client        *minio.Client
	bucketName    string
	externalHost  string
	archiveBucket string
	logger        *slog.Logger
}

type UploadResult struct {
	URL      string
	Key      string
	Type     string
	Size     int64
	MimeType string
}

func NewStorageService(cfg *config.Config, logger *slog.Logger) (*StorageService, error) {
	if logger == nil {
		logger = slog.Default()
	}

	minioClient, err := minio.New(cfg.StorageEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.StorageAccessKey, cfg.StorageSecretKey, ""),
		Secure: cfg.StorageUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, cfg.StorageBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := minioClient.MakeBucket(ctx, cfg.StorageBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info("Created private bucket", "bucket", cfg.StorageBucket)
		// NOTE: Bucket is PRIVATE by default. Use GetPresignedURL for read access.
	}

	return &StorageService{
		client:        minioClient,
		bucketName:    cfg.StorageBucket,
		externalHost:  cfg.StoragePublicURL,
		archiveBucket: cfg.ArchiveBucket,
		logger:        logger,
	}, nil
}

func (s *StorageService) Upload(ctx context.Context, data io.Reader, size int64, filename, contentType string) (*UploadResult, error) {
	ext := filepath.Ext(filename)
	objectName := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), uuid.New().String(), ext)

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	mediaType := "file"
	if strings.HasPrefix(contentType, "image/") {
		mediaType = "image"
	} else if strings.HasPrefix(contentType, "video/") {
		mediaType = "video"
	} else if strings.HasPrefix(contentType, "audio/") {
		mediaType = "audio"
	}

	info, err := s.client.PutObject(ctx, s.bucketName, objectName, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s", s.externalHost, s.bucketName, info.Key)

	s.logger.Info("File uploaded", "key", info.Key, "size", info.Size)

	return &UploadResult{
		URL:      url,
		Key:      info.Key,
		Type:     mediaType,
		Size:     info.Size,
		MimeType: contentType,
	}, nil
}

func (s *StorageService) UploadMultiple(ctx context.Context, files []FileUpload) ([]*UploadResult, error) {
	var wg sync.WaitGroup
	results := make([]*UploadResult, len(files))
	errors := make([]error, len(files))

	for i, file := range files {
		wg.Add(1)
		go func(idx int, f FileUpload) {
			defer wg.Done()
			result, err := s.Upload(ctx, f.Data, f.Size, f.Filename, f.ContentType)
			if err != nil {
				errors[idx] = err
				return
			}
			results[idx] = result
		}(i, file)
	}

	wg.Wait()

	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

type FileUpload struct {
	Data        io.Reader
	Size        int64
	Filename    string
	ContentType string
}

func (s *StorageService) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	s.logger.Info("File deleted", "key", key)
	return nil
}

func (s *StorageService) DeleteByURL(ctx context.Context, fileURL string) error {
	key := extractKeyFromURL(fileURL, s.bucketName)
	if key == "" {
		return fmt.Errorf("invalid file URL")
	}
	return s.Delete(ctx, key)
}

func extractKeyFromURL(fileURL, bucketName string) string {
	prefix := "/" + bucketName + "/"
	idx := strings.Index(fileURL, prefix)
	if idx == -1 {
		return ""
	}
	return fileURL[idx+len(prefix):]
}

func (s *StorageService) UploadArchive(ctx context.Context, objectPath string, data []byte) error {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(data); err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip: %w", err)
	}

	exists, err := s.client.BucketExists(ctx, s.archiveBucket)
	if err != nil {
		return fmt.Errorf("failed to check archive bucket: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.archiveBucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create archive bucket: %w", err)
		}
	}

	_, err = s.client.PutObject(ctx, s.archiveBucket, objectPath, &buf, int64(buf.Len()), minio.PutObjectOptions{
		ContentType:     "application/gzip",
		ContentEncoding: "gzip",
	})
	if err != nil {
		return fmt.Errorf("failed to upload archive: %w", err)
	}

	s.logger.Info("Archive uploaded", "path", objectPath)
	return nil
}

func (s *StorageService) DownloadArchive(ctx context.Context, objectPath string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.archiveBucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get archive: %w", err)
	}
	defer obj.Close()

	gzReader, err := gzip.NewReader(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	data, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}

	return data, nil
}

func (s *StorageService) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

func (s *StorageService) GetPresignedUploadURL(ctx context.Context, filename, contentType, hash string, size int64) (uploadURL, publicURL, key string, isDuplicate bool, err error) {
	if hash == "" {
		return "", "", "", false, fmt.Errorf("hash is required for deduplication")
	}

	ext := filepath.Ext(filename)
	// Deduplication: Key is based on HASH, not random UUID.
	// We use a prefix to keep organization.
	objectKey := fmt.Sprintf("uploads/%s%s", hash, ext)
	publicURL = fmt.Sprintf("%s/%s/%s", s.externalHost, s.bucketName, objectKey)

	// 1. Check if file already exists (Deduplication)
	_, err = s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
	if err == nil {
		// Object exists! Return public URL and indicate duplicate.
		return "", publicURL, objectKey, true, nil
	}

	// 2. Generate Presigned PUT URL
	expiry := 15 * time.Minute
	u, err := s.client.PresignedPutObject(ctx, s.bucketName, objectKey, expiry)
	if err != nil {
		return "", "", "", false, fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}

	return u.String(), publicURL, objectKey, false, nil
}

func detectMediaType(contentType string) string {
	if strings.HasPrefix(contentType, "image/") {
		return "image"
	} else if strings.HasPrefix(contentType, "video/") {
		return "video"
	} else if strings.HasPrefix(contentType, "audio/") {
		return "audio"
	}
	return "file"
}
