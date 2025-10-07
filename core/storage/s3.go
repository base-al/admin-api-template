package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds configuration for S3 storage
type S3Config struct {
	APIKey          string
	APISecret       string
	AccessKeyID     string
	AccessKeySecret string
	AccountID       string
	Endpoint        string
	Bucket          string
	BaseURL         string
	Region          string
}

type s3Provider struct {
	client   *s3.Client
	bucket   string
	endpoint string
	baseURL  string
}

func NewS3Provider(config S3Config) (Provider, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "s3.amazonaws.com"
	}

	// Region default
	region := config.Region
	if region == "" {
		region = "us-east-1"
	}

	// Load AWS config
	var cfg aws.Config
	var err error

	cfg, err = awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.AccessKeySecret,
			"",
		)),
		awsconfig.WithRegion(region),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create S3 client with path-style addressing
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &s3Provider{
		client:   client,
		bucket:   config.Bucket,
		endpoint: endpoint,
		baseURL:  config.BaseURL,
	}, nil
}

func (p *s3Provider) Upload(file *multipart.FileHeader, config UploadConfig) (*UploadResult, error) {
	// Open source file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	filename := generateUniqueFilename(file.Filename)
	key := fmt.Sprintf("%s/%s", config.UploadPath, filename)

	// Upload to S3
	// Note: ACL is deprecated - use bucket policies instead for access control
	_, err = p.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
		Body:   src,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	return &UploadResult{
		Filename: filename,
		Path:     key,
		Size:     file.Size,
	}, nil
}

func (p *s3Provider) UploadBytes(data []byte, filename string, config UploadConfig) (*UploadResult, error) {
	// Generate unique filename
	uniqueFilename := generateUniqueFilename(filename)
	key := fmt.Sprintf("%s/%s", config.UploadPath, uniqueFilename)

	// Upload to S3
	// Note: ACL is deprecated - use bucket policies instead for access control
	_, err := p.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	return &UploadResult{
		Filename: uniqueFilename,
		Path:     key,
		Size:     int64(len(data)),
	}, nil
}

func (p *s3Provider) Delete(path string) error {
	_, err := p.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	return err
}

func (p *s3Provider) GetURL(path string) string {
	return fmt.Sprintf("https://%s/%s/%s", p.endpoint, p.bucket, path)
}
