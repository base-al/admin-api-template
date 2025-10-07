package media

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gorm.io/gorm"

	"base/core/storage"
)

// SyncResult represents the result of a sync operation
type SyncResult struct {
	TotalFiles      int      `json:"total_files"`
	ProcessedFiles  int      `json:"processed_files"`
	SkippedFiles    int      `json:"skipped_files"`
	FailedFiles     int      `json:"failed_files"`
	CreatedFolders  int      `json:"created_folders"`
	Errors          []string `json:"errors,omitempty"`
	DurationSeconds float64  `json:"duration_seconds"`
}

// R2Syncer handles syncing R2 bucket contents to media database
type R2Syncer struct {
	db          *gorm.DB
	s3Client    *s3.Client
	bucket      string
	cdnURL      string
	folderCache map[string]uint // Cache folder IDs by path
}

// NewR2Syncer creates a new R2 syncer
func NewR2Syncer(db *gorm.DB, storageProvider storage.Provider, bucket, cdnURL string) (*R2Syncer, error) {
	// Try to get S3 client from provider
	// This is a bit hacky but works for now
	r2Provider, ok := storageProvider.(*storage.R2Provider)
	if !ok {
		return nil, fmt.Errorf("storage provider is not R2")
	}

	return &R2Syncer{
		db:          db,
		s3Client:    r2Provider.GetClient(),
		bucket:      bucket,
		cdnURL:      strings.TrimRight(cdnURL, "/"),
		folderCache: make(map[string]uint),
	}, nil
}

// SyncFromR2 syncs files from R2 bucket to media database
func (s *R2Syncer) SyncFromR2(prefix string) (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{}

	// List all objects in bucket
	objects, err := s.listR2Objects(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list R2 objects: %w", err)
	}

	result.TotalFiles = len(objects)

	// Process each object
	for _, obj := range objects {
		key := aws.ToString(obj.Key)
		size := aws.ToInt64(obj.Size)

		// Skip folder markers
		if strings.HasSuffix(key, "/") {
			continue
		}

		// Strip prefix to get relative path
		relativeKey := key
		if prefix != "" {
			relativeKey = strings.TrimPrefix(key, prefix)
			relativeKey = strings.TrimPrefix(relativeKey, "/")
		}

		// Check if already exists
		if s.attachmentExists(key) {
			result.SkippedFiles++
			continue
		}

		// Process file
		if err := s.processFile(key, relativeKey, size); err != nil {
			result.FailedFiles++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", relativeKey, err))
			continue
		}

		result.ProcessedFiles++
	}

	result.CreatedFolders = len(s.folderCache)
	result.DurationSeconds = time.Since(startTime).Seconds()

	return result, nil
}

// listR2Objects lists all objects in R2 bucket with given prefix
func (s *R2Syncer) listR2Objects(prefix string) ([]types.Object, error) {
	var allObjects []types.Object

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	// Use the new paginator API
	paginator := s3.NewListObjectsV2Paginator(s.s3Client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, err
		}
		allObjects = append(allObjects, page.Contents...)
	}

	return allObjects, nil
}

// attachmentExists checks if attachment already exists for this path
func (s *R2Syncer) attachmentExists(path string) bool {
	var count int64
	s.db.Model(&storage.Attachment{}).Where("path = ?", path).Count(&count)
	return count > 0
}

// processFile processes a single file and creates media + attachment records
func (s *R2Syncer) processFile(key, relativeKey string, size int64) error {
	filename := filepath.Base(relativeKey)
	dirPath := filepath.Dir(relativeKey)

	// Get media type from extension
	mediaType := getMediaTypeFromExtension(filename)

	// Get or create folder hierarchy
	var parentID *uint
	if dirPath != "." && dirPath != "/" && dirPath != "" {
		folderID, err := s.ensureFolderHierarchy(dirPath)
		if err != nil {
			return fmt.Errorf("failed to create folder hierarchy: %w", err)
		}
		parentID = &folderID
	}

	// Create media name (clean filename without extension)
	mediaName := strings.TrimSuffix(filename, filepath.Ext(filename))
	mediaName = strings.ReplaceAll(mediaName, "_", " ")
	mediaName = strings.TrimSpace(mediaName)

	// Get immediate folder name
	immediateFolder := ""
	if dirPath != "." && dirPath != "/" && dirPath != "" {
		immediateFolder = filepath.Base(dirPath)
	}

	// Build CDN URL
	cdnURL := fmt.Sprintf("%s/%s", s.cdnURL, key)

	// Create metadata JSON
	metadataStr := fmt.Sprintf(`{"original_filename": "%s", "path": "%s", "size": %d}`,
		strings.ReplaceAll(filename, `"`, `\"`),
		strings.ReplaceAll(key, `"`, `\"`),
		size)

	// Create media record
	media := &Media{
		Name:        mediaName,
		Type:        mediaType,
		Folder:      immediateFolder,
		ParentId:    parentID,
		Description: "",
		Metadata:    &metadataStr,
	}

	if err := s.db.Create(media).Error; err != nil {
		return fmt.Errorf("failed to create media record: %w", err)
	}

	// Create attachment record
	attachment := &storage.Attachment{
		ModelType: "media",
		ModelId:   media.Id,
		Field:     "file",
		Filename:  filename,
		Path:      key,
		Size:      size,
		URL:       cdnURL,
	}

	if err := s.db.Create(attachment).Error; err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	// Update media with file reference (JSON field)
	fileJSON := fmt.Sprintf(`{"id": %d, "filename": "%s", "path": "%s", "size": %d, "url": "%s"}`,
		attachment.Id, filename, key, size, cdnURL)

	if err := s.db.Model(media).Update("file", gorm.Expr("?", fileJSON)).Error; err != nil {
		return fmt.Errorf("failed to update media with file reference: %w", err)
	}

	return nil
}

// ensureFolderHierarchy ensures all parent folders exist and returns the leaf folder ID
func (s *R2Syncer) ensureFolderHierarchy(path string) (uint, error) {
	// Check cache first
	if id, exists := s.folderCache[path]; exists {
		return id, nil
	}

	// Split path into parts
	parts := strings.Split(filepath.ToSlash(path), "/")
	currentPath := ""
	var parentID *uint

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Build current path
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = filepath.Join(currentPath, part)
		}

		// Check if folder exists in cache
		if id, exists := s.folderCache[currentPath]; exists {
			parentID = &id
			continue
		}

		// Check if folder exists in database
		var folder Media
		query := s.db.Where("type = ? AND folder = ?", "folder", currentPath)
		if err := query.First(&folder).Error; err == nil {
			// Folder exists
			s.folderCache[currentPath] = folder.Id
			parentID = &folder.Id
			continue
		}

		// Create folder with metadata
		folderMetadataStr := fmt.Sprintf(`{"path": "%s/"}`, strings.ReplaceAll(currentPath, `"`, `\"`))

		folder = Media{
			Name:        part,
			Type:        "folder",
			Folder:      currentPath,
			ParentId:    parentID,
			Description: "",
			Metadata:    &folderMetadataStr,
		}

		if err := s.db.Create(&folder).Error; err != nil {
			return 0, fmt.Errorf("failed to create folder %s: %w", currentPath, err)
		}

		s.folderCache[currentPath] = folder.Id
		parentID = &folder.Id
	}

	if parentID == nil {
		return 0, fmt.Errorf("failed to get folder ID for path: %s", path)
	}

	return *parentID, nil
}

// getMediaTypeFromExtension determines media type from file extension
func getMediaTypeFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".tif":
		return "image"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac":
		return "audio"
	case ".mp4", ".webm", ".mov", ".avi", ".mkv":
		return "video"
	default:
		return "other"
	}
}
