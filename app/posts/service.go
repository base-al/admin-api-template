package posts

import (
	"math"
	"mime/multipart"

	"base/app/models"
	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"
	"base/core/types"

	"gorm.io/gorm"
)

const (
	CreatePostEvent = "posts.create"
	UpdatePostEvent = "posts.update"
	DeletePostEvent = "posts.delete"
)

type PostService struct {
	DB      *gorm.DB
	Emitter *emitter.Emitter
	Storage *storage.ActiveStorage
	Logger  logger.Logger
}

func NewPostService(db *gorm.DB, emitter *emitter.Emitter, storage *storage.ActiveStorage, logger logger.Logger) *PostService {
	return &PostService{
		DB:      db,
		Logger:  logger,
		Emitter: emitter,
		Storage: storage,
	}
}

// applySorting applies sorting to the query based on the sort and order parameters
func (s *PostService) applySorting(query *gorm.DB, sortBy *string, sortOrder *string) {
	// Valid sortable fields for Post
	validSortFields := map[string]string{
		"id":             "id",
		"created_at":     "created_at",
		"updated_at":     "updated_at",
		"title":          "title",
		"slug":           "slug",
		"content":        "content",
		"excerpt":        "excerpt",
		"author_id":      "author_id",
		"status":         "status",
		"category":       "category",
		"featured_image": "featured_image",
		"published":      "published",
		"featured":       "featured",
		"is_pinned":      "is_pinned",
		"view_count":     "view_count",
		"like_count":     "like_count",
		"rating":         "rating",
		"published_at":   "published_at",
		"scheduled_at":   "scheduled_at",
		"tags":           "tags",
		"metadata":       "metadata",
	}

	// Default sorting - if sort_order exists, always use it for custom ordering
	defaultSortBy := "id"
	defaultSortOrder := "desc"

	// Determine sort field
	sortField := defaultSortBy
	if sortBy != nil && *sortBy != "" {
		if field, exists := validSortFields[*sortBy]; exists {
			sortField = field
		}
	}

	// Determine sort direction (order parameter)
	sortDirection := defaultSortOrder
	if sortOrder != nil && (*sortOrder == "asc" || *sortOrder == "desc") {
		sortDirection = *sortOrder
	}

	// Apply sorting
	query.Order(sortField + " " + sortDirection)
}

func (s *PostService) Create(req *models.CreatePostRequest) (*models.Post, error) {
	item := &models.Post{
		Title:    req.Title,
		Slug:     req.Slug,
		Content:  req.Content,
		Excerpt:  req.Excerpt,
		AuthorId: req.AuthorId,
		Status:   req.Status,
		Category: req.Category,
		// handled separately
		Published:   req.Published,
		Featured:    req.Featured,
		IsPinned:    req.IsPinned,
		ViewCount:   req.ViewCount,
		LikeCount:   req.LikeCount,
		Rating:      req.Rating,
		PublishedAt: req.PublishedAt,
		ScheduledAt: req.ScheduledAt,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
	}

	if err := s.DB.Create(item).Error; err != nil {
		s.Logger.Error("failed to create post", logger.String("error", err.Error()))
		return nil, err
	}

	// Emit create event
	s.Emitter.Emit(CreatePostEvent, item)

	return s.GetById(item.Id)
}

func (s *PostService) Update(id uint, req *models.UpdatePostRequest) (*models.Post, error) {
	item := &models.Post{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find post for update",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Validate request
	if err := ValidatePostUpdateRequest(req, id); err != nil {
		return nil, err
	}

	// Update fields directly on the model
	// For non-pointer string fields
	if req.Title != "" {
		item.Title = req.Title
	}
	// For non-pointer string fields
	if req.Slug != "" {
		item.Slug = req.Slug
	}
	// For non-pointer string fields
	if req.Content != "" {
		item.Content = req.Content
	}
	// For non-pointer string fields
	if req.Excerpt != "" {
		item.Excerpt = req.Excerpt
	}
	// For non-pointer unsigned integer fields
	if req.AuthorId != 0 {
		item.AuthorId = req.AuthorId
	}
	// For non-pointer string fields
	if req.Status != "" {
		item.Status = req.Status
	}
	// For non-pointer string fields
	if req.Category != "" {
		item.Category = req.Category
	}
	// FeaturedImage attachment is handled via separate endpoint
	// For boolean fields, check if it's included in the request (pointer would be non-nil)
	if req.Published != nil {
		item.Published = *req.Published
	}
	// For boolean fields, check if it's included in the request (pointer would be non-nil)
	if req.Featured != nil {
		item.Featured = *req.Featured
	}
	// For boolean fields, check if it's included in the request (pointer would be non-nil)
	if req.IsPinned != nil {
		item.IsPinned = *req.IsPinned
	}
	// For non-pointer integer fields
	if req.ViewCount != 0 {
		item.ViewCount = req.ViewCount
	}
	// For non-pointer integer fields
	if req.LikeCount != 0 {
		item.LikeCount = req.LikeCount
	}
	// For non-pointer float fields
	if req.Rating != 0 {
		item.Rating = req.Rating
	}
	// For custom DateTime fields
	if !req.PublishedAt.IsZero() {
		item.PublishedAt = req.PublishedAt
	}
	// For custom DateTime fields
	if !req.ScheduledAt.IsZero() {
		item.ScheduledAt = req.ScheduledAt
	}

	if err := s.DB.Save(item).Error; err != nil {
		s.Logger.Error("failed to update post",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Handle many-to-many relationships

	result, err := s.GetById(item.Id)
	if err != nil {
		s.Logger.Error("failed to get updated post",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Emit update event
	s.Emitter.Emit(UpdatePostEvent, result)

	return result, nil
}

func (s *PostService) Delete(id uint) error {
	item := &models.Post{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find post for deletion",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Delete file attachments if any
	if item.FeaturedImage != nil {
		if err := s.Storage.Delete(item.FeaturedImage); err != nil {
			s.Logger.Error("failed to delete featured_image",
				logger.String("error", err.Error()),
				logger.Int("id", int(id)))
			return err
		}
	}

	if err := s.DB.Delete(item).Error; err != nil {
		s.Logger.Error("failed to delete post",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Emit delete event
	s.Emitter.Emit(DeletePostEvent, item)

	return nil
}

func (s *PostService) GetById(id uint) (*models.Post, error) {
	item := &models.Post{}

	query := item.Preload(s.DB)
	if err := query.First(item, id).Error; err != nil {
		s.Logger.Error("failed to get post",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	return item, nil
}

func (s *PostService) GetAll(page *int, limit *int, sortBy *string, sortOrder *string) (*types.PaginatedResponse, error) {
	var items []*models.Post
	var total int64

	query := s.DB.Model(&models.Post{})
	// Set default values if nil
	defaultPage := 1
	defaultLimit := 10
	if page == nil {
		page = &defaultPage
	}
	if limit == nil {
		limit = &defaultLimit
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.Logger.Error("failed to count posts",
			logger.String("error", err.Error()))
		return nil, err
	}

	// Apply pagination if provided
	if page != nil && limit != nil {
		offset := (*page - 1) * *limit
		query = query.Offset(offset).Limit(*limit)
	}

	// Apply sorting
	s.applySorting(query, sortBy, sortOrder)

	// Don't preload relationships for list response (faster)
	// query = (&models.Post{}).Preload(query)

	// Execute query
	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("failed to get posts",
			logger.String("error", err.Error()))
		return nil, err
	}

	// Convert to response type
	responses := make([]*models.PostListResponse, len(items))
	for i, item := range items {
		responses[i] = item.ToListResponse()
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(*limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &types.PaginatedResponse{
		Data: responses,
		Pagination: types.Pagination{
			Total:      int(total),
			Page:       *page,
			PageSize:   *limit,
			TotalPages: totalPages,
		},
	}, nil
}

// GetAllForSelect gets all items for select box/dropdown options (simplified response)
func (s *PostService) GetAllForSelect() ([]*models.Post, error) {
	var items []*models.Post

	query := s.DB.Model(&models.Post{})

	// Only select the necessary fields for select options
	query = query.Select("id, title")

	// Order by name/title for better UX
	query = query.Order("title ASC")

	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("Failed to fetch items for select", logger.String("error", err.Error()))
		return nil, err
	}

	return items, nil
}

// UploadFeaturedImage uploads a file for the Post's FeaturedImage field
func (s *PostService) UploadFeaturedImage(id uint, file *multipart.FileHeader) (*models.Post, error) {
	item := &models.Post{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find post",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Delete existing file if any
	if item.FeaturedImage != nil {
		if err := s.Storage.Delete(item.FeaturedImage); err != nil {
			s.Logger.Error("failed to delete existing featured_image",
				logger.String("error", err.Error()),
				logger.Int("id", int(id)))
			return nil, err
		}
	}

	// Attach new file
	attachment, err := s.Storage.Attach(item, "featured_image", file)
	if err != nil {
		s.Logger.Error("failed to attach featured_image",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Update the model with the new attachment
	if err := s.DB.Model(item).Association("FeaturedImage").Replace(attachment); err != nil {
		s.Logger.Error("failed to associate featured_image",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	return s.GetById(id)
}

// RemoveFeaturedImage removes the file from the Post's FeaturedImage field
func (s *PostService) RemoveFeaturedImage(id uint) (*models.Post, error) {
	item := &models.Post{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find post",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	if item.FeaturedImage == nil {
		return item, nil
	}

	if err := s.Storage.Delete(item.FeaturedImage); err != nil {
		s.Logger.Error("failed to delete featured_image",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Clear the association
	if err := s.DB.Model(item).Association("FeaturedImage").Clear(); err != nil {
		s.Logger.Error("failed to clear featured_image association",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	return s.GetById(id)
}
