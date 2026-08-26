package repository

import (
	"context"
	"io"
)

//go:generate go tool mockgen -destination=../mock/mock_repository/uploader.go golang-s3-batch-uploader/repository Uploader
type Uploader interface {
	Upload(ctx context.Context, key string, body io.Reader) error
}
