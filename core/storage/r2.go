package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Config holds configuration for Cloudflare R2 storage
type R2Config struct {
	AccessKeyID     string
	AccessKeySecret string
	AccountID       string
	Bucket          string
	BaseURL         string
	CDN             string
}

type r2Provider struct {
	client   *s3.Client
	bucket   string
	endpoint string
	baseURL  string
	cdn      string
}

// R2Provider is the exported type for type assertions
type R2Provider = r2Provider

// GetClient returns the S3 client (for internal use like syncing)
func (p *r2Provider) GetClient() *s3.Client {
	return p.client
}

func NewR2Provider(config R2Config) (Provider, error) {
	// R2 endpoint format: https://<account_id>.r2.cloudflarestorage.com
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", config.AccountID)

	// Load AWS config with custom resolver
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.AccessKeySecret,
			"",
		)),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create R2 config: %w", err)
	}

	// Create S3 client with path-style addressing
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &r2Provider{
		client:   client,
		bucket:   config.Bucket,
		endpoint: endpoint,
		baseURL:  config.BaseURL,
		cdn:      config.CDN,
	}, nil
}

func (p *r2Provider) Upload(file *multipart.FileHeader, config UploadConfig) (*UploadResult, error) {
	// Open source file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	filename := generateUniqueFilename(file.Filename)
	key := fmt.Sprintf("%s/%s", config.UploadPath, filename)

	// Upload to R2
	_, err = p.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		Body:        src,
		ContentType: aws.String(file.Header.Get("Content-Type")),
		// Note: R2 doesn't support ACL, so we remove the ACL setting
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to R2: %w", err)
	}

	return &UploadResult{
		Filename: filename,
		Path:     key,
		Size:     file.Size,
	}, nil
}

func (p *r2Provider) UploadBytes(data []byte, filename string, config UploadConfig) (*UploadResult, error) {
	// Generate unique filename
	uniqueFilename := generateUniqueFilename(filename)
	key := fmt.Sprintf("%s/%s", config.UploadPath, uniqueFilename)

	// Detect content type from filename
	contentType := "application/octet-stream"
	if strings.HasSuffix(strings.ToLower(filename), ".webp") {
		contentType = "image/webp"
	} else if strings.HasSuffix(strings.ToLower(filename), ".webm") {
		contentType = "video/webm"
	}

	// Upload to R2
	_, err := p.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to R2: %w", err)
	}

	return &UploadResult{
		Filename: uniqueFilename,
		Path:     key,
		Size:     int64(len(data)),
	}, nil
}

func (p *r2Provider) Delete(path string) error {
	_, err := p.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
	})
	return err
}

func (p *r2Provider) GetURL(path string) string {
	// Always prefer CDN for R2 storage
	if p.cdn != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(p.cdn, "/"), path)
	}
	// Fallback to BaseURL if CDN is not configured
	if p.baseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(p.baseURL, "/"), path)
	}
	// Last resort: use R2 URL
	return fmt.Sprintf("https://%s/%s/%s", p.endpoint, p.bucket, path)
}
