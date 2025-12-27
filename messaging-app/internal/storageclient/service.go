package storageclient

import (
	"context"
	"io"
	"time"

	storagepb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/storage/v1"
)

func (c *Client) Upload(ctx context.Context, data []byte, filename, contentType string) (*UploadResult, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.Upload(ctx, &storagepb.UploadRequest{
			Data:        data,
			Filename:    filename,
			ContentType: contentType,
		})
	})
	if err != nil {
		return nil, err
	}

	return ToUploadResult(result.(*storagepb.UploadResponse)), nil
}

func (c *Client) UploadFromReader(ctx context.Context, reader io.Reader, filename, contentType string) (*UploadResult, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return c.Upload(ctx, data, filename, contentType)
}

func (c *Client) UploadMultiple(ctx context.Context, files []FileUploadRequest) ([]*UploadResult, error) {
	pbFiles := make([]*storagepb.FileUpload, len(files))
	for i, f := range files {
		pbFiles[i] = &storagepb.FileUpload{
			Data:        f.Data,
			Filename:    f.Filename,
			ContentType: f.ContentType,
		}
	}

	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UploadMultiple(ctx, &storagepb.UploadMultipleRequest{
			Files: pbFiles,
		})
	})
	if err != nil {
		return nil, err
	}

	return ToUploadResults(result.(*storagepb.UploadMultipleResponse).Results), nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.Delete(ctx, &storagepb.DeleteRequest{
			Key: key,
		})
	})
	return err
}

func (c *Client) DeleteByURL(ctx context.Context, url string) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DeleteByURL(ctx, &storagepb.DeleteByURLRequest{
			Url: url,
		})
	})
	return err
}

func (c *Client) UploadArchive(ctx context.Context, objectPath string, data []byte) error {
	_, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.UploadArchive(ctx, &storagepb.UploadArchiveRequest{
			ObjectPath: objectPath,
			Data:       data,
		})
	})
	return err
}

func (c *Client) DownloadArchive(ctx context.Context, objectPath string) ([]byte, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.DownloadArchive(ctx, &storagepb.DownloadArchiveRequest{
			ObjectPath: objectPath,
		})
	})
	if err != nil {
		return nil, err
	}
	return result.(*storagepb.DownloadArchiveResponse).Data, nil
}

func (c *Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	result, err := c.cb.Execute(ctx, func() (interface{}, error) {
		return c.client.GetPresignedURL(ctx, &storagepb.GetPresignedURLRequest{
			Key:           key,
			ExpirySeconds: int64(expiry.Seconds()),
		})
	})
	if err != nil {
		return "", err
	}
	return result.(*storagepb.GetPresignedURLResponse).Url, nil
}
