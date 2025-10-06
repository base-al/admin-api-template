package activities

import (
	"encoding/json"
	"math"

	"base/app/models"
	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"
	"base/core/types"

	"gorm.io/gorm"
)

const (
	CreateActivityEvent = "activities.create"
	UpdateActivityEvent = "activities.update"
	DeleteActivityEvent = "activities.delete"
)

type ActivityService struct {
	DB      *gorm.DB
	Emitter *emitter.Emitter
	Storage *storage.ActiveStorage
	Logger  logger.Logger
}

func NewActivityService(db *gorm.DB, emitter *emitter.Emitter, storage *storage.ActiveStorage, logger logger.Logger) *ActivityService {
	return &ActivityService{
		DB:      db,
		Logger:  logger,
		Emitter: emitter,
		Storage: storage,
	}
}

// applySorting applies sorting to the query based on the sort and order parameters
func (s *ActivityService) applySorting(query *gorm.DB, sortBy *string, sortOrder *string) {
	// Valid sortable fields for Activity
	validSortFields := map[string]string{
		"id":          "id",
		"created_at":  "created_at",
		"updated_at":  "updated_at",
		"user_id":     "user_id",
		"entity_type": "entity_type",
		"entity_id":   "entity_id",
		"action":      "action",
		"description": "description",
		"metadata":    "metadata",
		"ip_address":  "ip_address",
		"user_agent":  "user_agent",
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

func (s *ActivityService) Create(req *models.CreateActivityRequest) (*models.Activity, error) {
	item := &models.Activity{
		UserId:      req.UserId,
		EntityType:  req.EntityType,
		EntityId:    req.EntityId,
		Action:      req.Action,
		Description: req.Description,
		Metadata:    req.Metadata,
		IpAddress:   req.IpAddress,
		UserAgent:   req.UserAgent,
	}

	if err := s.DB.Create(item).Error; err != nil {
		s.Logger.Error("failed to create activity", logger.String("error", err.Error()))
		return nil, err
	}

	// Emit create event
	s.Emitter.Emit(CreateActivityEvent, item)

	return s.GetById(item.Id)
}

func (s *ActivityService) Update(id uint, req *models.UpdateActivityRequest) (*models.Activity, error) {
	item := &models.Activity{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find activity for update",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Validate request
	if err := ValidateActivityUpdateRequest(req, id); err != nil {
		return nil, err
	}

	// Update fields directly on the model
	// For non-pointer unsigned integer fields
	if req.UserId != 0 {
		item.UserId = req.UserId
	}
	// For non-pointer string fields
	if req.EntityType != "" {
		item.EntityType = req.EntityType
	}
	// For non-pointer unsigned integer fields
	if req.EntityId != 0 {
		item.EntityId = req.EntityId
	}
	// For non-pointer string fields
	if req.Action != "" {
		item.Action = req.Action
	}
	// For non-pointer string fields
	if req.Description != "" {
		item.Description = req.Description
	}
	// For non-pointer string fields
	if req.IpAddress != "" {
		item.IpAddress = req.IpAddress
	}
	// For non-pointer string fields
	if req.UserAgent != "" {
		item.UserAgent = req.UserAgent
	}

	if err := s.DB.Save(item).Error; err != nil {
		s.Logger.Error("failed to update activity",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Handle many-to-many relationships

	result, err := s.GetById(item.Id)
	if err != nil {
		s.Logger.Error("failed to get updated activity",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Emit update event
	s.Emitter.Emit(UpdateActivityEvent, result)

	return result, nil
}

func (s *ActivityService) Delete(id uint) error {
	item := &models.Activity{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find activity for deletion",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Delete file attachments if any

	if err := s.DB.Delete(item).Error; err != nil {
		s.Logger.Error("failed to delete activity",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Emit delete event
	s.Emitter.Emit(DeleteActivityEvent, item)

	return nil
}

func (s *ActivityService) GetById(id uint) (*models.Activity, error) {
	item := &models.Activity{}

	query := item.Preload(s.DB)
	if err := query.First(item, id).Error; err != nil {
		s.Logger.Error("failed to get activity",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	return item, nil
}

func (s *ActivityService) GetAll(page *int, limit *int, sortBy *string, sortOrder *string) (*types.PaginatedResponse, error) {
	var items []*models.Activity
	var total int64

	query := s.DB.Model(&models.Activity{})
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
		s.Logger.Error("failed to count activities",
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
	// query = (&models.Activity{}).Preload(query)

	// Execute query
	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("failed to get activities",
			logger.String("error", err.Error()))
		return nil, err
	}

	// Convert to response type
	responses := make([]*models.ActivityListResponse, len(items))
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
func (s *ActivityService) GetAllForSelect() ([]*models.Activity, error) {
	var items []*models.Activity

	query := s.DB.Model(&models.Activity{})

	// Only select the necessary fields for select options
	query = query.Select("id") // Only ID if no name/title field found

	// Order by name/title for better UX
	query = query.Order("id ASC")

	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("Failed to fetch items for select", logger.String("error", err.Error()))
		return nil, err
	}

	return items, nil
}

// Log is a convenient helper to log an activity
// Example: Log(userId, "post", postId, "create", "Created new blog post", metadata, ipAddress, userAgent)
func (s *ActivityService) Log(userId uint, entityType string, entityId uint, action string, description string, metadata map[string]interface{}, ipAddress string, userAgent string) error {
	var metadataJSON []byte
	var err error

	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			s.Logger.Error("failed to marshal activity metadata", logger.String("error", err.Error()))
			return err
		}
	}

	req := &models.CreateActivityRequest{
		UserId:      userId,
		EntityType:  entityType,
		EntityId:    entityId,
		Action:      action,
		Description: description,
		Metadata:    metadataJSON,
		IpAddress:   ipAddress,
		UserAgent:   userAgent,
	}

	_, err = s.Create(req)
	if err != nil {
		s.Logger.Error("failed to log activity", logger.String("error", err.Error()))
		return err
	}

	return nil
}

// GetRecentActivities gets the most recent N activities
func (s *ActivityService) GetRecentActivities(limit int) ([]*models.Activity, error) {
	var activities []*models.Activity

	query := s.DB.Model(&models.Activity{}).
		Order("created_at DESC").
		Limit(limit)

	// Preload user relationship
	query = (&models.Activity{}).Preload(query)

	if err := query.Find(&activities).Error; err != nil {
		s.Logger.Error("failed to get recent activities", logger.String("error", err.Error()))
		return nil, err
	}

	return activities, nil
}

// GetActivitiesByUser gets activities for a specific user
func (s *ActivityService) GetActivitiesByUser(userId uint, limit int) ([]*models.Activity, error) {
	var activities []*models.Activity

	query := s.DB.Model(&models.Activity{}).
		Where("user_id = ?", userId).
		Order("created_at DESC").
		Limit(limit)

	query = (&models.Activity{}).Preload(query)

	if err := query.Find(&activities).Error; err != nil {
		s.Logger.Error("failed to get user activities", logger.String("error", err.Error()))
		return nil, err
	}

	return activities, nil
}

// GetActivitiesByEntity gets activities for a specific entity
func (s *ActivityService) GetActivitiesByEntity(entityType string, entityId uint, limit int) ([]*models.Activity, error) {
	var activities []*models.Activity

	query := s.DB.Model(&models.Activity{}).
		Where("entity_type = ? AND entity_id = ?", entityType, entityId).
		Order("created_at DESC").
		Limit(limit)

	query = (&models.Activity{}).Preload(query)

	if err := query.Find(&activities).Error; err != nil {
		s.Logger.Error("failed to get entity activities", logger.String("error", err.Error()))
		return nil, err
	}

	return activities, nil
}
