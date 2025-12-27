package grpc

import (
	"bytes"
	"context"
	"time"

	storagepb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/storage/v1"
	"github.com/MuhibNayem/connectify-v2/storage-service/internal/service"
)

type StorageHandler struct {
	storagepb.UnimplementedStorageServiceServer
	svc *service.StorageService
}

func NewStorageHandler(svc *service.StorageService) *StorageHandler {
	return &StorageHandler{svc: svc}
}

func (h *StorageHandler) Upload(ctx context.Context, req *storagepb.UploadRequest) (*storagepb.UploadResponse, error) {
	result, err := h.svc.Upload(ctx, bytes.NewReader(req.Data), int64(len(req.Data)), req.Filename, req.ContentType)
	if err != nil {
		return nil, err
	}
	return &storagepb.UploadResponse{
		Url:      result.URL,
		Key:      result.Key,
		Type:     result.Type,
		Size:     result.Size,
		MimeType: result.MimeType,
	}, nil
}

func (h *StorageHandler) UploadMultiple(ctx context.Context, req *storagepb.UploadMultipleRequest) (*storagepb.UploadMultipleResponse, error) {
	files := make([]service.FileUpload, len(req.Files))
	for i, f := range req.Files {
		files[i] = service.FileUpload{
			Data:        bytes.NewReader(f.Data),
			Size:        int64(len(f.Data)),
			Filename:    f.Filename,
			ContentType: f.ContentType,
		}
	}

	results, err := h.svc.UploadMultiple(ctx, files)
	if err != nil {
		return nil, err
	}

	pbResults := make([]*storagepb.UploadResponse, len(results))
	for i, r := range results {
		pbResults[i] = &storagepb.UploadResponse{
			Url:      r.URL,
			Key:      r.Key,
			Type:     r.Type,
			Size:     r.Size,
			MimeType: r.MimeType,
		}
	}

	return &storagepb.UploadMultipleResponse{Results: pbResults}, nil
}

func (h *StorageHandler) Delete(ctx context.Context, req *storagepb.DeleteRequest) (*storagepb.DeleteResponse, error) {
	err := h.svc.Delete(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	return &storagepb.DeleteResponse{Success: true}, nil
}

func (h *StorageHandler) DeleteByURL(ctx context.Context, req *storagepb.DeleteByURLRequest) (*storagepb.DeleteResponse, error) {
	err := h.svc.DeleteByURL(ctx, req.Url)
	if err != nil {
		return nil, err
	}
	return &storagepb.DeleteResponse{Success: true}, nil
}

func (h *StorageHandler) UploadArchive(ctx context.Context, req *storagepb.UploadArchiveRequest) (*storagepb.UploadArchiveResponse, error) {
	err := h.svc.UploadArchive(ctx, req.ObjectPath, req.Data)
	if err != nil {
		return nil, err
	}
	return &storagepb.UploadArchiveResponse{Success: true}, nil
}

func (h *StorageHandler) DownloadArchive(ctx context.Context, req *storagepb.DownloadArchiveRequest) (*storagepb.DownloadArchiveResponse, error) {
	data, err := h.svc.DownloadArchive(ctx, req.ObjectPath)
	if err != nil {
		return nil, err
	}
	return &storagepb.DownloadArchiveResponse{Data: data}, nil
}

func (h *StorageHandler) GetPresignedURL(ctx context.Context, req *storagepb.GetPresignedURLRequest) (*storagepb.GetPresignedURLResponse, error) {
	expiry := time.Duration(req.ExpirySeconds) * time.Second
	if expiry == 0 {
		expiry = 15 * time.Minute
	}
	url, err := h.svc.GetPresignedURL(ctx, req.Key, expiry)
	if err != nil {
		return nil, err
	}
	return &storagepb.GetPresignedURLResponse{Url: url}, nil
}
