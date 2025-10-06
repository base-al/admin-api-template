package notifications

import (
	"math"

	"base/app/models"
	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"
	"base/core/types"

	"gorm.io/gorm"
)

const (
	CreateNotificationEvent = "notifications.create"
	UpdateNotificationEvent = "notifications.update"
	DeleteNotificationEvent = "notifications.delete"
)

type NotificationService struct {
	DB      *gorm.DB
	Emitter *emitter.Emitter
	Storage *storage.ActiveStorage
	Logger  logger.Logger
}

func NewNotificationService(db *gorm.DB, emitter *emitter.Emitter, storage *storage.ActiveStorage, logger logger.Logger) *NotificationService {
	return &NotificationService{
		DB:      db,
		Logger:  logger,
		Emitter: emitter,
		Storage: storage,
	}
}

// applySorting applies sorting to the query based on the sort and order parameters
func (s *NotificationService) applySorting(query *gorm.DB, sortBy *string, sortOrder *string) {
	// Valid sortable fields for Notification
	validSortFields := map[string]string{
		"id":         "id",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"user_id":    "user_id",
		"title":      "title",
		"body":       "body",
		"type":       "type",
		"read":       "read",
		"read_at":    "read_at",
		"action_url": "action_url",
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

func (s *NotificationService) Create(req *models.CreateNotificationRequest) (*models.Notification, error) {
	item := &models.Notification{
		UserId:    req.UserId,
		Title:     req.Title,
		Body:      req.Body,
		Type:      req.Type,
		Read:      req.Read,
		ReadAt:    req.ReadAt,
		ActionUrl: req.ActionUrl,
	}

	if err := s.DB.Create(item).Error; err != nil {
		s.Logger.Error("failed to create notification", logger.String("error", err.Error()))
		return nil, err
	}

	// Emit create event
	s.Emitter.Emit(CreateNotificationEvent, item)

	return s.GetById(item.Id)
}

func (s *NotificationService) Update(id uint, req *models.UpdateNotificationRequest) (*models.Notification, error) {
	item := &models.Notification{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find notification for update",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Validate request
	if err := ValidateNotificationUpdateRequest(req, id); err != nil {
		return nil, err
	}

	// Update fields directly on the model
	// For non-pointer unsigned integer fields
	if req.UserId != 0 {
		item.UserId = req.UserId
	}
	// For non-pointer string fields
	if req.Title != "" {
		item.Title = req.Title
	}
	// For non-pointer string fields
	if req.Body != "" {
		item.Body = req.Body
	}
	// For non-pointer string fields
	if req.Type != "" {
		item.Type = req.Type
	}
	// For boolean fields, check if it's included in the request (pointer would be non-nil)
	if req.Read != nil {
		item.Read = *req.Read
	}
	// For custom DateTime fields
	if !req.ReadAt.IsZero() {
		item.ReadAt = req.ReadAt
	}
	// For non-pointer string fields
	if req.ActionUrl != "" {
		item.ActionUrl = req.ActionUrl
	}

	if err := s.DB.Save(item).Error; err != nil {
		s.Logger.Error("failed to update notification",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Handle many-to-many relationships

	result, err := s.GetById(item.Id)
	if err != nil {
		s.Logger.Error("failed to get updated notification",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Emit update event
	s.Emitter.Emit(UpdateNotificationEvent, result)

	return result, nil
}

func (s *NotificationService) Delete(id uint) error {
	item := &models.Notification{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find notification for deletion",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Delete file attachments if any

	if err := s.DB.Delete(item).Error; err != nil {
		s.Logger.Error("failed to delete notification",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Emit delete event
	s.Emitter.Emit(DeleteNotificationEvent, item)

	return nil
}

func (s *NotificationService) GetById(id uint) (*models.Notification, error) {
	item := &models.Notification{}

	query := item.Preload(s.DB)
	if err := query.First(item, id).Error; err != nil {
		s.Logger.Error("failed to get notification",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	return item, nil
}

func (s *NotificationService) GetAll(page *int, limit *int, sortBy *string, sortOrder *string) (*types.PaginatedResponse, error) {
	var items []*models.Notification
	var total int64

	query := s.DB.Model(&models.Notification{})
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
		s.Logger.Error("failed to count notifications",
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
	// query = (&models.Notification{}).Preload(query)

	// Execute query
	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("failed to get notifications",
			logger.String("error", err.Error()))
		return nil, err
	}

	// Convert to response type
	responses := make([]*models.NotificationListResponse, len(items))
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
func (s *NotificationService) GetAllForSelect() ([]*models.Notification, error) {
	var items []*models.Notification

	query := s.DB.Model(&models.Notification{})

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
