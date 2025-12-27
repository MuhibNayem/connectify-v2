package storageclient

import (
	storagepb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/storage/v1"
)

type UploadResult struct {
	URL      string
	Key      string
	Type     string
	Size     int64
	MimeType string
}

type FileUploadRequest struct {
	Data        []byte
	Filename    string
	ContentType string
}

func ToUploadResult(pb *storagepb.UploadResponse) *UploadResult {
	if pb == nil {
		return nil
	}
	return &UploadResult{
		URL:      pb.Url,
		Key:      pb.Key,
		Type:     pb.Type,
		Size:     pb.Size,
		MimeType: pb.MimeType,
	}
}

func ToUploadResults(pbs []*storagepb.UploadResponse) []*UploadResult {
	results := make([]*UploadResult, 0, len(pbs))
	for _, pb := range pbs {
		if r := ToUploadResult(pb); r != nil {
			results = append(results, r)
		}
	}
	return results
}
