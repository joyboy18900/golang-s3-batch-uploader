package repository

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type uploaderS3 struct {
	client *s3.Client
	bucket string
}

func NewUploaderS3(ctx context.Context, region, endpoint, bucket string, autoCreateBucket bool) (Uploader, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})

	u := uploaderS3{client: client, bucket: bucket}
	if autoCreateBucket {
		if err := u.ensureBucket(ctx); err != nil {
			return nil, err
		}
	}

	return u, nil
}

func (u uploaderS3) ensureBucket(ctx context.Context) error {
	if _, err := u.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(u.bucket)}); err == nil {
		return nil
	}

	_, err := u.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(u.bucket)})

	var alreadyOwned *types.BucketAlreadyOwnedByYou
	var alreadyExists *types.BucketAlreadyExists
	if err != nil && !errors.As(err, &alreadyOwned) && !errors.As(err, &alreadyExists) {
		return err
	}
	return nil
}

func (u uploaderS3) Upload(ctx context.Context, key string, body io.Reader) error {
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	return err
}
