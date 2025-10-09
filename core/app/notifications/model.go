package notifications

import (
	"base/core/types"
	"time"

	"gorm.io/gorm"
)

// Notification represents a notification entity
type Notification struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	UserId    uint           `json:"user_id"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Type      string         `json:"type"`
	Read      bool           `json:"read"`
	ReadAt    types.DateTime `json:"read_at"`
	ActionUrl string         `json:"action_url"`
}

// TableName returns the table name for the Notification model
func (m *Notification) TableName() string {
	return "notifications"
}

// GetId returns the Id of the model
func (m *Notification) GetId() uint {
	return m.Id
}

// GetModelName returns the model name
func (m *Notification) GetModelName() string {
	return "notification"
}

// CreateNotificationRequest represents the request payload for creating a Notification
type CreateNotificationRequest struct {
	UserId    uint           `json:"user_id"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Type      string         `json:"type"`
	Read      bool           `json:"read"`
	ReadAt    types.DateTime `json:"read_at" swaggertype:"string"`
	ActionUrl string         `json:"action_url"`
}

// UpdateNotificationRequest represents the request payload for updating a Notification
type UpdateNotificationRequest struct {
	UserId    uint           `json:"user_id,omitempty"`
	Title     string         `json:"title,omitempty"`
	Body      string         `json:"body,omitempty"`
	Type      string         `json:"type,omitempty"`
	Read      *bool          `json:"read,omitempty"`
	ReadAt    types.DateTime `json:"read_at,omitempty" swaggertype:"string"`
	ActionUrl string         `json:"action_url,omitempty"`
}

// NotificationResponse represents the API response for Notification
type NotificationResponse struct {
	Id        uint           `json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`
	UserId    uint           `json:"user_id"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Type      string         `json:"type"`
	Read      bool           `json:"read"`
	ReadAt    types.DateTime `json:"read_at"`
	ActionUrl string         `json:"action_url"`
}

// NotificationModelResponse represents a simplified response when this model is part of other entities
type NotificationModelResponse struct {
	Id    uint   `json:"id"`
	Title string `json:"title"`
}

// NotificationSelectOption represents a simplified response for select boxes and dropdowns
type NotificationSelectOption struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // From Title field
}

// NotificationListResponse represents the response for list operations (optimized for performance)
type NotificationListResponse struct {
	Id        uint           `json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`
	UserId    uint           `json:"user_id"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Type      string         `json:"type"`
	Read      bool           `json:"read"`
	ReadAt    types.DateTime `json:"read_at"`
	ActionUrl string         `json:"action_url"`
}

// ToResponse converts the model to an API response
func (m *Notification) ToResponse() *NotificationResponse {
	if m == nil {
		return nil
	}
	response := &NotificationResponse{
		Id:        m.Id,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
		UserId:    m.UserId,
		Title:     m.Title,
		Body:      m.Body,
		Type:      m.Type,
		Read:      m.Read,
		ReadAt:    m.ReadAt,
		ActionUrl: m.ActionUrl,
	}

	return response
}

// ToModelResponse converts the model to a simplified response for when it's part of other entities
func (m *Notification) ToModelResponse() *NotificationModelResponse {
	if m == nil {
		return nil
	}
	return &NotificationModelResponse{
		Id:    m.Id,
		Title: m.Title,
	}
}

// ToSelectOption converts the model to a select option for dropdowns
func (m *Notification) ToSelectOption() *NotificationSelectOption {
	if m == nil {
		return nil
	}
	displayName := m.Title

	return &NotificationSelectOption{
		Id:   m.Id,
		Name: displayName,
	}
}

// ToListResponse converts the model to a list response (without preloaded relationships for fast listing)
func (m *Notification) ToListResponse() *NotificationListResponse {
	if m == nil {
		return nil
	}
	return &NotificationListResponse{
		Id:        m.Id,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
		UserId:    m.UserId,
		Title:     m.Title,
		Body:      m.Body,
		Type:      m.Type,
		Read:      m.Read,
		ReadAt:    m.ReadAt,
		ActionUrl: m.ActionUrl,
	}
}

// Preload preloads all the model's relationships
func (m *Notification) Preload(db *gorm.DB) *gorm.DB {
	query := db
	return query
}
