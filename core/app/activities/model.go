package activities

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"base/core/app/users"
)

// Activity represents an audit log entry tracking system actions
type Activity struct {
	Id        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at" gorm:"index"` // Indexed for fast sorting
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// User who performed the action
	UserId uint          `json:"user_id" gorm:"index"` // Indexed for filtering by user
	User   *users.User `json:"user,omitempty" gorm:"foreignKey:UserId"`

	// Entity being acted upon (e.g., "post", "employee", "order")
	EntityType string `json:"entity_type" gorm:"index"` // Indexed for filtering by entity
	EntityId   uint   `json:"entity_id" gorm:"index"`   // Indexed for filtering by entity ID

	// Action performed (e.g., "create", "update", "delete", "login", "logout")
	Action string `json:"action" gorm:"index"` // Indexed for filtering by action

	// Human-readable description (e.g., "Created new post", "Updated employee profile")
	Description string `json:"description"`

	// Additional context (old/new values, etc.)
	Metadata json.RawMessage `json:"metadata" gorm:"type:json"`

	// Request context
	IpAddress string `json:"ip_address" gorm:"index"` // Indexed for security auditing
	UserAgent string `json:"user_agent"`
}

// TableName returns the table name for the Activity model
func (m *Activity) TableName() string {
	return "activities"
}

// GetId returns the Id of the model
func (m *Activity) GetId() uint {
	return m.Id
}

// GetModelName returns the model name
func (m *Activity) GetModelName() string {
	return "activity"
}

// CreateActivityRequest represents the request payload for creating a Activity
type CreateActivityRequest struct {
	UserId      uint            `json:"user_id"`
	EntityType  string          `json:"entity_type"`
	EntityId    uint            `json:"entity_id"`
	Action      string          `json:"action"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	IpAddress   string          `json:"ip_address"`
	UserAgent   string          `json:"user_agent"`
}

// UpdateActivityRequest represents the request payload for updating a Activity
type UpdateActivityRequest struct {
	UserId      uint            `json:"user_id,omitempty"`
	EntityType  string          `json:"entity_type,omitempty"`
	EntityId    uint            `json:"entity_id,omitempty"`
	Action      string          `json:"action,omitempty"`
	Description string          `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	IpAddress   string          `json:"ip_address,omitempty"`
	UserAgent   string          `json:"user_agent,omitempty"`
}

// ActivityResponse represents the API response for Activity
type ActivityResponse struct {
	Id          uint                       `json:"id"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	DeletedAt   gorm.DeletedAt             `json:"deleted_at"`
	UserId      uint                       `json:"user_id"`
	User        *users.UserModelResponse `json:"user,omitempty"`
	EntityType  string                     `json:"entity_type"`
	EntityId    uint                       `json:"entity_id"`
	Action      string                     `json:"action"`
	Description string                     `json:"description"`
	Metadata    json.RawMessage            `json:"metadata"`
	IpAddress   string                     `json:"ip_address"`
	UserAgent   string                     `json:"user_agent"`
}

// ActivityModelResponse represents a simplified response when this model is part of other entities
type ActivityModelResponse struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // Display name
}

// ActivitySelectOption represents a simplified response for select boxes and dropdowns
type ActivitySelectOption struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // Display name
}

// ActivityListResponse represents the response for list operations (optimized for performance)
type ActivityListResponse struct {
	Id          uint            `json:"id"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `json:"deleted_at"`
	UserId      uint            `json:"user_id"`
	EntityType  string          `json:"entity_type"`
	EntityId    uint            `json:"entity_id"`
	Action      string          `json:"action"`
	Description string          `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	IpAddress   string          `json:"ip_address"`
	UserAgent   string          `json:"user_agent"`
}

// ToResponse converts the model to an API response
func (m *Activity) ToResponse() *ActivityResponse {
	if m == nil {
		return nil
	}
	response := &ActivityResponse{
		Id:          m.Id,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
		UserId:      m.UserId,
		EntityType:  m.EntityType,
		EntityId:    m.EntityId,
		Action:      m.Action,
		Description: m.Description,
		Metadata:    m.Metadata,
		IpAddress:   m.IpAddress,
		UserAgent:   m.UserAgent,
	}

	// Include user if loaded
	if m.User != nil {
		response.User = m.User.ToModelResponse()
	}

	return response
}

// ToModelResponse converts the model to a simplified response for when it's part of other entities
func (m *Activity) ToModelResponse() *ActivityModelResponse {
	if m == nil {
		return nil
	}
	return &ActivityModelResponse{
		Id:   m.Id,
		Name: fmt.Sprintf("Activity #%d", m.Id), // Fallback to ID-based display
	}
}

// ToSelectOption converts the model to a select option for dropdowns
func (m *Activity) ToSelectOption() *ActivitySelectOption {
	if m == nil {
		return nil
	}
	displayName := m.EntityType // Using first string field as display name

	return &ActivitySelectOption{
		Id:   m.Id,
		Name: displayName,
	}
}

// ToListResponse converts the model to a list response (without preloaded relationships for fast listing)
func (m *Activity) ToListResponse() *ActivityListResponse {
	if m == nil {
		return nil
	}
	return &ActivityListResponse{
		Id:          m.Id,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
		UserId:      m.UserId,
		EntityType:  m.EntityType,
		EntityId:    m.EntityId,
		Action:      m.Action,
		Description: m.Description,
		Metadata:    m.Metadata,
		IpAddress:   m.IpAddress,
		UserAgent:   m.UserAgent,
	}
}

// Preload preloads all the model's relationships
func (m *Activity) Preload(db *gorm.DB) *gorm.DB {
	query := db.Preload("User")
	return query
}
