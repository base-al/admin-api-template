package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Settings represents a settings entity
type Settings struct {
	Id          uint           `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	SettingKey  string         `json:"setting_key" gorm:"type:varchar(100);index"`
	Label       string         `json:"label" gorm:"type:varchar(200)"`
	Group       string         `json:"group" gorm:"type:varchar(50)"`
	Type        string         `json:"type" gorm:"type:varchar(20)"`
	ValueString string         `json:"value_string" gorm:"type:text"`
	ValueInt    int            `json:"value_int"`
	ValueFloat  float64        `json:"value_float"`
	ValueBool   bool           `json:"value_bool"`
	Description string         `json:"description" gorm:"type:text"`
	IsPublic    bool           `json:"is_public"`
}

// TableName returns the table name for the Settings model
func (m *Settings) TableName() string {
	return "settings"
}

// GetId returns the Id of the model
func (m *Settings) GetId() uint {
	return m.Id
}

// GetModelName returns the model name
func (m *Settings) GetModelName() string {
	return "settings"
}

// CreateSettingsRequest represents the request payload for creating a Settings
type CreateSettingsRequest struct {
	SettingKey  string  `json:"setting_key"`
	Label       string  `json:"label"`
	Group       string  `json:"group"`
	Type        string  `json:"type"`
	ValueString string  `json:"value_string"`
	ValueInt    int     `json:"value_int"`
	ValueFloat  float64 `json:"value_float"`
	ValueBool   bool    `json:"value_bool"`
	Description string  `json:"description"`
	IsPublic    bool    `json:"is_public"`
}

// UpdateSettingsRequest represents the request payload for updating a Settings
type UpdateSettingsRequest struct {
	SettingKey  string  `json:"setting_key,omitempty"`
	Label       string  `json:"label,omitempty"`
	Group       string  `json:"group,omitempty"`
	Type        string  `json:"type,omitempty"`
	ValueString string  `json:"value_string,omitempty"`
	ValueInt    int     `json:"value_int,omitempty"`
	ValueFloat  float64 `json:"value_float,omitempty"`
	ValueBool   *bool   `json:"value_bool,omitempty"`
	Description string  `json:"description,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
}

// SettingsResponse represents the API response for Settings
type SettingsResponse struct {
	Id          uint           `json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at"`
	SettingKey  string         `json:"setting_key"`
	Label       string         `json:"label"`
	Group       string         `json:"group"`
	Type        string         `json:"type"`
	ValueString string         `json:"value_string"`
	ValueInt    int            `json:"value_int"`
	ValueFloat  float64        `json:"value_float"`
	ValueBool   bool           `json:"value_bool"`
	Description string         `json:"description"`
	IsPublic    bool           `json:"is_public"`
}

// SettingsModelResponse represents a simplified response when this model is part of other entities
type SettingsModelResponse struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // Display name
}

// SettingsSelectOption represents a simplified response for select boxes and dropdowns
type SettingsSelectOption struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // Display name
}

// SettingsListResponse represents the response for list operations (optimized for performance)
type SettingsListResponse struct {
	Id          uint           `json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at"`
	SettingKey  string         `json:"setting_key"`
	Label       string         `json:"label"`
	Group       string         `json:"group"`
	Type        string         `json:"type"`
	ValueString string         `json:"value_string"`
	ValueInt    int            `json:"value_int"`
	ValueFloat  float64        `json:"value_float"`
	ValueBool   bool           `json:"value_bool"`
	Description string         `json:"description"`
	IsPublic    bool           `json:"is_public"`
}

// ToResponse converts the model to an API response
func (m *Settings) ToResponse() *SettingsResponse {
	if m == nil {
		return nil
	}
	response := &SettingsResponse{
		Id:          m.Id,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
		SettingKey:  m.SettingKey,
		Label:       m.Label,
		Group:       m.Group,
		Type:        m.Type,
		ValueString: m.ValueString,
		ValueInt:    m.ValueInt,
		ValueFloat:  m.ValueFloat,
		ValueBool:   m.ValueBool,
		Description: m.Description,
		IsPublic:    m.IsPublic,
	}

	return response
}

// ToModelResponse converts the model to a simplified response for when it's part of other entities
func (m *Settings) ToModelResponse() *SettingsModelResponse {
	if m == nil {
		return nil
	}
	return &SettingsModelResponse{
		Id:   m.Id,
		Name: fmt.Sprintf("Settings #%d", m.Id), // Fallback to ID-based display
	}
}

// ToSelectOption converts the model to a select option for dropdowns
func (m *Settings) ToSelectOption() *SettingsSelectOption {
	if m == nil {
		return nil
	}
	displayName := m.SettingKey // Using first string field as display name

	return &SettingsSelectOption{
		Id:   m.Id,
		Name: displayName,
	}
}

// ToListResponse converts the model to a list response (without preloaded relationships for fast listing)
func (m *Settings) ToListResponse() *SettingsListResponse {
	if m == nil {
		return nil
	}
	return &SettingsListResponse{
		Id:          m.Id,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
		SettingKey:  m.SettingKey,
		Label:       m.Label,
		Group:       m.Group,
		Type:        m.Type,
		ValueString: m.ValueString,
		ValueInt:    m.ValueInt,
		ValueFloat:  m.ValueFloat,
		ValueBool:   m.ValueBool,
		Description: m.Description,
		IsPublic:    m.IsPublic,
	}
}

// Preload preloads all the model's relationships
func (m *Settings) Preload(db *gorm.DB) *gorm.DB {
	query := db
	return query
}
