package settings

import (
	"fmt"
	"math"

	"base/app/models"
	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"
	"base/core/types"

	"gorm.io/gorm"
)

const (
	CreateSettingsEvent = "settings.create"
	UpdateSettingsEvent = "settings.update"
	DeleteSettingsEvent = "settings.delete"
)

type SettingsService struct {
	DB      *gorm.DB
	Emitter *emitter.Emitter
	Storage *storage.ActiveStorage
	Logger  logger.Logger
}

func NewSettingsService(db *gorm.DB, emitter *emitter.Emitter, storage *storage.ActiveStorage, logger logger.Logger) *SettingsService {
	return &SettingsService{
		DB:      db,
		Logger:  logger,
		Emitter: emitter,
		Storage: storage,
	}
}

// Configuration helper methods for modules to retrieve settings

// GetSettingString retrieves a string setting value by key
func (s *SettingsService) GetSettingString(key string, defaultValue string) string {
	var setting models.Settings
	if err := s.DB.Where("setting_key = ?", key).First(&setting).Error; err != nil {
		s.Logger.Warn("setting not found, using default", 
			logger.String("key", key), 
			logger.String("default", defaultValue))
		return defaultValue
	}
	return setting.ValueString
}

// GetSettingInt retrieves an integer setting value by key
func (s *SettingsService) GetSettingInt(key string, defaultValue int) int {
	var setting models.Settings
	if err := s.DB.Where("setting_key = ?", key).First(&setting).Error; err != nil {
		s.Logger.Warn("setting not found, using default", 
			logger.String("key", key), 
			logger.Int("default", defaultValue))
		return defaultValue
	}
	return setting.ValueInt
}

// GetSettingBool retrieves a boolean setting value by key
func (s *SettingsService) GetSettingBool(key string, defaultValue bool) bool {
	var setting models.Settings
	if err := s.DB.Where("setting_key = ?", key).First(&setting).Error; err != nil {
		s.Logger.Warn("setting not found, using default", 
			logger.String("key", key), 
			logger.Bool("default", defaultValue))
		return defaultValue
	}
	return setting.ValueBool
}

// GetSettingFloat retrieves a float setting value by key
func (s *SettingsService) GetSettingFloat(key string, defaultValue float64) float64 {
	var setting models.Settings
	if err := s.DB.Where("setting_key = ?", key).First(&setting).Error; err != nil {
		s.Logger.Warn("setting not found, using default", 
			logger.String("key", key), 
			logger.Float64("default", defaultValue))
		return defaultValue
	}
	return setting.ValueFloat
}

// GetSettingsByGroup retrieves all settings for a specific group
func (s *SettingsService) GetSettingsByGroup(group string) ([]*models.Settings, error) {
	var settings []*models.Settings
	if err := s.DB.Where("group = ?", group).Find(&settings).Error; err != nil {
		s.Logger.Error("failed to get settings by group", 
			logger.String("group", group), 
			logger.String("error", err.Error()))
		return nil, err
	}
	return settings, nil
}

// UpsertSetting creates or updates a setting
func (s *SettingsService) UpsertSetting(key, label, group, settingType, description string, isPublic bool, stringVal string, intVal int, floatVal float64, boolVal bool) error {
	var setting models.Settings
	result := s.DB.Where("setting_key = ?", key).First(&setting)
	
	if result.Error != nil && result.Error.Error() == "record not found" {
		// Create new setting
		setting = models.Settings{
			SettingKey:  key,
			Label:       label,
			Group:       group,
			Type:        settingType,
			Description: description,
			IsPublic:    isPublic,
			ValueString: stringVal,
			ValueInt:    intVal,
			ValueFloat:  floatVal,
			ValueBool:   boolVal,
		}
		if err := s.DB.Create(&setting).Error; err != nil {
			s.Logger.Error("failed to create setting", 
				logger.String("key", key), 
				logger.String("error", err.Error()))
			return err
		}
	} else if result.Error == nil {
		// Update existing setting
		setting.Label = label
		setting.Group = group
		setting.Type = settingType
		setting.Description = description
		setting.IsPublic = isPublic
		setting.ValueString = stringVal
		setting.ValueInt = intVal
		setting.ValueFloat = floatVal
		setting.ValueBool = boolVal
		
		if err := s.DB.Save(&setting).Error; err != nil {
			s.Logger.Error("failed to update setting", 
				logger.String("key", key), 
				logger.String("error", err.Error()))
			return err
		}
	} else {
		return result.Error
	}
	
	return nil
}

// applySorting applies sorting to the query based on the sort and order parameters
func (s *SettingsService) applySorting(query *gorm.DB, sortBy *string, sortOrder *string) {
	// Valid sortable fields for Settings
	validSortFields := map[string]string{
		"id":           "id",
		"created_at":   "created_at",
		"updated_at":   "updated_at",
		"setting_key":  "setting_key",
		"label":        "label",
		"group":        "group",
		"type":         "type",
		"value_string": "value_string",
		"value_int":    "value_int",
		"value_float":  "value_float",
		"value_bool":   "value_bool",
		"description":  "description",
		"is_public":    "is_public",
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

func (s *SettingsService) Create(req *models.CreateSettingsRequest) (*models.Settings, error) {
	item := &models.Settings{
		SettingKey:  req.SettingKey,
		Label:       req.Label,
		Group:       req.Group,
		Type:        req.Type,
		ValueString: req.ValueString,
		ValueInt:    req.ValueInt,
		ValueFloat:  req.ValueFloat,
		ValueBool:   req.ValueBool,
		Description: req.Description,
		IsPublic:    req.IsPublic,
	}

	if err := s.DB.Create(item).Error; err != nil {
		s.Logger.Error("failed to create settings", logger.String("error", err.Error()))
		return nil, err
	}

	// Emit create event
	s.Emitter.Emit(CreateSettingsEvent, item)

	return s.GetById(item.Id)
}

func (s *SettingsService) Update(id uint, req *models.UpdateSettingsRequest) (*models.Settings, error) {
	item := &models.Settings{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find settings for update",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Validate request
	if err := ValidateSettingsUpdateRequest(req, id); err != nil {
		return nil, err
	}

	// Update fields directly on the model
	// For non-pointer string fields
	if req.SettingKey != "" {
		item.SettingKey = req.SettingKey
	}
	// For non-pointer string fields
	if req.Label != "" {
		item.Label = req.Label
	}
	// For non-pointer string fields
	if req.Group != "" {
		item.Group = req.Group
	}
	// For non-pointer string fields
	if req.Type != "" {
		item.Type = req.Type
	}
	// For non-pointer string fields
	if req.ValueString != "" {
		item.ValueString = req.ValueString
	}
	// For non-pointer integer fields
	if req.ValueInt != 0 {
		item.ValueInt = req.ValueInt
	}
	// For boolean fields, check if it's included in the request (pointer would be non-nil)
	if req.ValueBool != nil {
		item.ValueBool = *req.ValueBool
	}
	// For non-pointer string fields
	if req.Description != "" {
		item.Description = req.Description
	}
	// For boolean fields, check if it's included in the request (pointer would be non-nil)
	if req.IsPublic != nil {
		item.IsPublic = *req.IsPublic
	}

	if err := s.DB.Save(item).Error; err != nil {
		s.Logger.Error("failed to update settings",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Handle many-to-many relationships

	result, err := s.GetById(item.Id)
	if err != nil {
		s.Logger.Error("failed to get updated settings",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Emit update event
	s.Emitter.Emit(UpdateSettingsEvent, result)

	return result, nil
}

func (s *SettingsService) Delete(id uint) error {
	item := &models.Settings{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find settings for deletion",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Delete file attachments if any

	if err := s.DB.Delete(item).Error; err != nil {
		s.Logger.Error("failed to delete settings",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Emit delete event
	s.Emitter.Emit(DeleteSettingsEvent, item)

	return nil
}

func (s *SettingsService) GetById(id uint) (*models.Settings, error) {
	item := &models.Settings{}

	query := item.Preload(s.DB)
	if err := query.First(item, id).Error; err != nil {
		s.Logger.Error("failed to get settings",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	return item, nil
}

func (s *SettingsService) GetAll(page *int, limit *int, sortBy *string, sortOrder *string) (*types.PaginatedResponse, error) {
	var items []*models.Settings
	var total int64

	query := s.DB.Model(&models.Settings{})
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
		s.Logger.Error("failed to count settings",
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
	// query = (&models.Settings{}).Preload(query)

	// Execute query
	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("failed to get settings",
			logger.String("error", err.Error()))
		return nil, err
	}

	// Convert to response type
	responses := make([]*models.SettingsListResponse, len(items))
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
func (s *SettingsService) GetAllForSelect() ([]*models.Settings, error) {
	var items []*models.Settings

	query := s.DB.Model(&models.Settings{})

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

// GetByKey retrieves a setting value by its setting_key
func (s *SettingsService) GetByKey(settingKey string) (*models.Settings, error) {
	item := &models.Settings{}
	if err := s.DB.Where("setting_key = ?", settingKey).First(item).Error; err != nil {
		s.Logger.Error("failed to get setting by key",
			logger.String("error", err.Error()),
			logger.String("setting_key", settingKey))
		return nil, err
	}
	return item, nil
}

// GetByGroup retrieves all settings in a group
func (s *SettingsService) GetByGroup(group string) ([]*models.Settings, error) {
	var items []*models.Settings
	if err := s.DB.Where("group = ?", group).Find(&items).Error; err != nil {
		s.Logger.Error("failed to get settings by group",
			logger.String("error", err.Error()),
			logger.String("group", group))
		return nil, err
	}
	return items, nil
}

// GetStringValue gets a string setting value by key
func (s *SettingsService) GetStringValue(settingKey string, defaultValue string) string {
	setting, err := s.GetByKey(settingKey)
	if err != nil {
		return defaultValue
	}
	if setting.Type != "string" {
		return defaultValue
	}
	return setting.ValueString
}

// GetIntValue gets an integer setting value by key
func (s *SettingsService) GetIntValue(settingKey string, defaultValue int) int {
	setting, err := s.GetByKey(settingKey)
	if err != nil {
		return defaultValue
	}
	if setting.Type != "int" {
		return defaultValue
	}
	return setting.ValueInt
}

// GetFloatValue gets a float setting value by key
func (s *SettingsService) GetFloatValue(settingKey string, defaultValue float64) float64 {
	setting, err := s.GetByKey(settingKey)
	if err != nil {
		return defaultValue
	}
	if setting.Type != "float" {
		return defaultValue
	}
	return setting.ValueFloat
}

// GetBoolValue gets a boolean setting value by key
func (s *SettingsService) GetBoolValue(settingKey string, defaultValue bool) bool {
	setting, err := s.GetByKey(settingKey)
	if err != nil {
		return defaultValue
	}
	if setting.Type != "bool" {
		return defaultValue
	}
	return setting.ValueBool
}

// SetStringValue creates or updates a string setting
func (s *SettingsService) SetStringValue(settingKey, label, group, description string, value string, isPublic bool) error {
	return s.SetValue(settingKey, label, group, "string", value, 0, 0.0, false, description, isPublic)
}

// SetIntValue creates or updates an integer setting
func (s *SettingsService) SetIntValue(settingKey, label, group, description string, value int, isPublic bool) error {
	return s.SetValue(settingKey, label, group, "int", "", value, 0.0, false, description, isPublic)
}

// SetFloatValue creates or updates a float setting
func (s *SettingsService) SetFloatValue(settingKey, label, group, description string, value float64, isPublic bool) error {
	return s.SetValue(settingKey, label, group, "float", "", 0, value, false, description, isPublic)
}

// SetBoolValue creates or updates a boolean setting
func (s *SettingsService) SetBoolValue(settingKey, label, group, description string, value bool, isPublic bool) error {
	return s.SetValue(settingKey, label, group, "bool", "", 0, 0.0, value, description, isPublic)
}

// SetValue creates or updates a setting value by key
func (s *SettingsService) SetValue(settingKey, label, group, settingType string, valueString string, valueInt int, valueFloat float64, valueBool bool, description string, isPublic bool) error {
	// Try to find existing setting
	existing, err := s.GetByKey(settingKey)
	if err != nil {
		// Create new setting
		_, createErr := s.Create(&models.CreateSettingsRequest{
			SettingKey:  settingKey,
			Label:       label,
			Group:       group,
			Type:        settingType,
			ValueString: valueString,
			ValueInt:    valueInt,
			ValueFloat:  valueFloat,
			ValueBool:   valueBool,
			Description: description,
			IsPublic:    isPublic,
		})
		return createErr
	}

	// Update existing setting
	_, updateErr := s.Update(existing.Id, &models.UpdateSettingsRequest{
		SettingKey:  settingKey,
		Label:       label,
		Group:       group,
		Type:        settingType,
		ValueString: valueString,
		ValueInt:    valueInt,
		ValueFloat:  valueFloat,
		ValueBool:   &valueBool,
		Description: description,
		IsPublic:    &isPublic,
	})
	return updateErr
}

// Get retrieves a setting value by key and returns any
func (s *SettingsService) Get(settingKey string, defaultValue any) any {
	setting, err := s.GetByKey(settingKey)
	if err != nil {
		return defaultValue
	}
	
	switch setting.Type {
	case "string":
		return setting.ValueString
	case "int":
		return setting.ValueInt
	case "float":
		return setting.ValueFloat
	case "bool":
		return setting.ValueBool
	default:
		return defaultValue
	}
}

// Set creates or updates a setting with automatic type detection
func (s *SettingsService) Set(settingKey, label, group, description string, value any, isPublic bool) error {
	switch v := value.(type) {
	case string:
		return s.SetStringValue(settingKey, label, group, description, v, isPublic)
	case int:
		return s.SetIntValue(settingKey, label, group, description, v, isPublic)
	case float64:
		return s.SetFloatValue(settingKey, label, group, description, v, isPublic)
	case bool:
		return s.SetBoolValue(settingKey, label, group, description, v, isPublic)
	default:
		return fmt.Errorf("unsupported setting type: %T", value)
	}
}
