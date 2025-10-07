package media

import (
	"context"
	"fmt"
	"math"
	"mime/multipart"

	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"
	"base/core/types"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MediaService struct {
	DB            *gorm.DB
	Emitter       *emitter.Emitter
	ActiveStorage *storage.ActiveStorage
	Logger        logger.Logger
}

func NewMediaService(db *gorm.DB, emitter *emitter.Emitter, activeStorage *storage.ActiveStorage, logger logger.Logger) *MediaService {
	// Register file attachment configuration
	// Note: Images (jpg, jpeg, png, heic, heif) will be auto-converted to webp
	// Videos (mp4, mov, avi, etc.) will be auto-converted to webm
	activeStorage.RegisterAttachment("media", storage.AttachmentConfig{
		Field:             "file",
		Path:              "media/files",
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".heic", ".heif", ".webp", ".mp4", ".mov", ".avi", ".mkv", ".webm", ".mp3", ".wav", ".ogg", ".opus"},
		MaxFileSize:       100 << 20, // 100MB
		Multiple:          false,
	})

	// Register original_file attachment configuration (for keeping originals)
	activeStorage.RegisterAttachment("media", storage.AttachmentConfig{
		Field:             "original_file",
		Path:              "media/files/originals",
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".heic", ".heif", ".webp", ".mp4", ".mov", ".avi", ".mkv", ".webm", ".mp3", ".wav", ".ogg", ".opus"},
		MaxFileSize:       100 << 20, // 100MB
		Multiple:          false,
	})

	return &MediaService{
		DB:            db,
		Emitter:       emitter,
		ActiveStorage: activeStorage,
		Logger:        logger,
	}
}

// GetById returns a single media item by id
func (s *MediaService) GetById(id uint) (*Media, error) {
	var item Media

	if err := s.DB.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("media not found")
		}
		s.Logger.Error("failed to get media", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	// Load relationships
	if err := s.DB.Preload(clause.Associations).First(&item, id).Error; err != nil {
		s.Logger.Error("failed to load media relationships", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to load media relationships: %w", err)
	}

	return &item, nil
}

// GetByIds returns multiple media items by their IDs
func (s *MediaService) GetByIds(ids []uint) ([]*Media, error) {
	if len(ids) == 0 {
		return []*Media{}, nil
	}

	var items []*Media
	if err := s.DB.Where("id IN ?", ids).Preload(clause.Associations).Find(&items).Error; err != nil {
		s.Logger.Error("failed to get media by ids", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get media by ids: %w", err)
	}

	return items, nil
}

// GetAll returns a paginated list of media items
func (s *MediaService) GetAll(page, limit *int) (*types.PaginatedResponse, error) {
	var items []*Media
	var total int64

	// Get total count
	if err := s.DB.Model(&Media{}).Count(&total).Error; err != nil {
		s.Logger.Error("failed to count media", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to count media: %w", err)
	}

	// Build query
	query := s.DB.Model(&Media{})

	// Add pagination if provided
	if page != nil && limit != nil {
		offset := (*page - 1) * *limit
		query = query.Offset(offset).Limit(*limit)
	}

	// Execute query with preloads
	if err := query.Preload(clause.Associations).Find(&items).Error; err != nil {
		s.Logger.Error("failed to get media", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	// Convert to response
	responses := make([]any, len(items))
	for i, item := range items {
		responses[i] = item.ToListResponse()
	}

	// Calculate pagination
	pageSize := 10
	currentPage := 1
	if limit != nil {
		pageSize = *limit
	}
	if page != nil {
		currentPage = *page
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	// Build paginated response
	return &types.PaginatedResponse{
		Data: responses,
		Pagination: types.Pagination{
			Total:      int(total),
			Page:       currentPage,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	}, nil
}

// Create creates a new media item
func (s *MediaService) Create(req *CreateMediaRequest) (*Media, error) {
	// Begin transaction
	tx := s.DB.Begin()
	if tx.Error != nil {
		s.Logger.Error("failed to begin transaction", logger.String("error", tx.Error.Error()))
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create media item
	item := &Media{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		ParentId:    req.ParentId,
		Folder:      req.Folder,
		Tags:        req.Tags,
		AuthorId:    req.AuthorId,
	}

	// Handle metadata - only set if not empty, otherwise leave as nil for NULL
	if req.Metadata != "" {
		item.Metadata = &req.Metadata
	}

	if err := tx.Create(item).Error; err != nil {
		tx.Rollback()
		s.Logger.Error("failed to create media", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create media: %w", err)
	}

	// Handle file upload if provided
	if req.File != nil {
		// Upload the file using storage system
		attachment, err := s.ActiveStorage.Attach(item, "file", req.File)
		if err != nil {
			tx.Rollback()
			s.Logger.Error("failed to upload file", logger.String("error", err.Error()))
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}

		// Update media with file information
		item.File = attachment
		if err := tx.Save(item).Error; err != nil {
			tx.Rollback()
			s.Logger.Error("failed to update media with file", logger.String("error", err.Error()))
			return nil, fmt.Errorf("failed to update media with file: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		s.Logger.Error("failed to commit transaction", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload item with relationships
	return s.GetById(item.Id)
}

// Update updates a media item
func (s *MediaService) Update(id uint, req *UpdateMediaRequest) (*Media, error) {
	// Begin transaction
	tx := s.DB.Begin()
	if tx.Error != nil {
		s.Logger.Error("failed to begin transaction", logger.String("error", tx.Error.Error()))
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get existing item
	item, err := s.GetById(id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Type != nil {
		item.Type = *req.Type
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.AuthorId != nil {
		item.AuthorId = req.AuthorId
	}

	// Handle file update if provided
	if req.File != nil {
		// Remove existing file if any
		if item.File != nil {
			if err := s.ActiveStorage.Delete(item.File); err != nil {
				tx.Rollback()
				s.Logger.Error("failed to delete existing file", logger.String("error", err.Error()))
				return nil, fmt.Errorf("failed to delete existing file: %w", err)
			}
		}

		// Upload new file
		attachment, err := s.ActiveStorage.Attach(item, "file", req.File)
		if err != nil {
			tx.Rollback()
			s.Logger.Error("failed to upload file", logger.String("error", err.Error()))
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}

		// Update media with new file information
		item.File = attachment
	}

	// Save changes
	if err := tx.Save(item).Error; err != nil {
		tx.Rollback()
		s.Logger.Error("failed to update media", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to update media: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		s.Logger.Error("failed to commit transaction", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload item with relationships
	return s.GetById(id)
}

// Delete deletes a media item
func (s *MediaService) Delete(id uint) error {
	// Get existing item
	item, err := s.GetById(id)
	if err != nil {
		return err
	}

	// Begin transaction
	tx := s.DB.Begin()
	if tx.Error != nil {
		s.Logger.Error("failed to begin transaction", logger.String("error", tx.Error.Error()))
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete the file if it exists
	if item.File != nil {
		if err := s.ActiveStorage.Delete(item.File); err != nil {
			s.Logger.Error("failed to delete file", logger.String("error", err.Error()))
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	// Delete the media item
	if err := tx.Delete(item).Error; err != nil {
		tx.Rollback()
		s.Logger.Error("failed to delete media", logger.String("error", err.Error()))
		return fmt.Errorf("failed to delete media: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		s.Logger.Error("failed to commit transaction", logger.String("error", err.Error()))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateFile updates the file of a media item
func (s *MediaService) UpdateFile(ctx context.Context, id uint, file *multipart.FileHeader) (*Media, error) {
	// Begin transaction
	tx := s.DB.Begin()
	if tx.Error != nil {
		s.Logger.Error("failed to begin transaction", logger.String("error", tx.Error.Error()))
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get existing item
	item, err := s.GetById(id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Remove existing file if any
	if item.File != nil {
		if err := s.ActiveStorage.Delete(item.File); err != nil {
			tx.Rollback()
			s.Logger.Error("failed to delete existing file", logger.String("error", err.Error()))
			return nil, fmt.Errorf("failed to delete existing file: %w", err)
		}
	}

	// Upload new file
	attachment, err := s.ActiveStorage.Attach(item, "file", file)
	if err != nil {
		tx.Rollback()
		s.Logger.Error("failed to upload file", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Update media with new file information
	item.File = attachment
	if err := tx.Save(item).Error; err != nil {
		tx.Rollback()
		s.Logger.Error("failed to update media with file", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to update media with file: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		s.Logger.Error("failed to commit transaction", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload item with relationships
	return s.GetById(id)
}

// RemoveFile removes the file from a media item
func (s *MediaService) RemoveFile(ctx context.Context, id uint) (*Media, error) {
	// Begin transaction
	tx := s.DB.Begin()
	if tx.Error != nil {
		s.Logger.Error("failed to begin transaction", logger.String("error", tx.Error.Error()))
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get existing item
	item, err := s.GetById(id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Remove file if exists
	if item.File != nil {
		if err := s.ActiveStorage.Delete(item.File); err != nil {
			tx.Rollback()
			s.Logger.Error("failed to delete file", logger.String("error", err.Error()))
			return nil, fmt.Errorf("failed to delete file: %w", err)
		}

		// Update media item
		item.File = nil
		if err := tx.Save(item).Error; err != nil {
			tx.Rollback()
			s.Logger.Error("failed to update media", logger.String("error", err.Error()))
			return nil, fmt.Errorf("failed to update media: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		s.Logger.Error("failed to commit transaction", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reload item with relationships
	return s.GetById(id)
}

// SyncFromR2 syncs media files from R2 bucket to database
func (s *MediaService) SyncFromR2(activeStorage *storage.ActiveStorage, bucket, cdnURL, prefix string) (*SyncResult, error) {
	// Get storage provider
	provider := activeStorage.GetProvider()

	// Create syncer
	syncer, err := NewR2Syncer(s.DB, provider, bucket, cdnURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create R2 syncer: %w", err)
	}

	// Run sync
	result, err := syncer.SyncFromR2(prefix)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetAllWithFilters returns a paginated list of media items with filtering support
func (s *MediaService) GetAllWithFilters(page, limit *int, filters *MediaFilters) (*types.PaginatedResponse, error) {
	var items []*Media
	var total int64

	// Build query
	query := s.DB.Model(&Media{})

	// Apply filters
	hasFilters := filters != nil && (filters.ParentId != nil || filters.Folder != "" || filters.Type != "" || filters.AuthorId != nil)

	if hasFilters {
		// Filter by parent ID for hierarchical navigation
		if filters.ParentId != nil {
			query = query.Where("parent_id = ?", *filters.ParentId)
		} else if filters.Folder == "" {
			// If no parent_id and no folder filter, show only root level items
			query = query.Where("parent_id IS NULL")
		}

		// Filter by folder path for backward compatibility
		if filters.Folder != "" && filters.ParentId == nil {
			query = query.Where("folder = ? OR folder LIKE ?", filters.Folder, filters.Folder+"/%")
		}

		// Filter by type
		if filters.Type != "" {
			query = query.Where("type LIKE ?", "%"+filters.Type+"%")
		}

		// Filter by author ID
		if filters.AuthorId != nil {
			if filters.IncludeShared {
				// Include both author-specific and shared (null author_id) media
				query = query.Where("author_id = ? OR author_id IS NULL", *filters.AuthorId)
			} else {
				// Only author-specific media
				query = query.Where("author_id = ?", *filters.AuthorId)
			}
		} else if !filters.IncludeShared {
			// If no author filter but include_shared is false, show only shared media
			query = query.Where("author_id IS NULL")
		}
	} else if page != nil && limit != nil {
		// Default: show root level items when no actual filters in paginated request
		query = query.Where("parent_id IS NULL")
	}
	// If no actual filters and no pagination (ListAll case), return all items without parent_id filter

	// Get total count with filters applied
	if err := query.Count(&total).Error; err != nil {
		s.Logger.Error("failed to count media with filters", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to count media: %w", err)
	}

	// Add pagination if provided
	if page != nil && limit != nil {
		offset := (*page - 1) * *limit
		query = query.Offset(offset).Limit(*limit)
	}

	// Execute query with preloads
	if err := query.Preload(clause.Associations).Find(&items).Error; err != nil {
		s.Logger.Error("failed to get media with filters", logger.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	// Convert to response
	responses := make([]any, len(items))
	for i, item := range items {
		responses[i] = item.ToListResponse()
	}

	// Calculate pagination
	pageSize := 10
	currentPage := 1
	if limit != nil {
		pageSize = *limit
	}
	if page != nil {
		currentPage = *page
	}
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	// Build paginated response
	return &types.PaginatedResponse{
		Data: responses,
		Pagination: types.Pagination{
			Total:      int(total),
			Page:       currentPage,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	}, nil
}
