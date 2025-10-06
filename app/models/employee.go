package models

import (
	"base/core/app/authorization"
	"fmt"
	"time"

	"gorm.io/gorm"
)


// Employee represents a employee entity
type Employee struct {
	Id        uint                `json:"id" gorm:"primarykey"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	DeletedAt gorm.DeletedAt      `json:"deleted_at" gorm:"index"`
	FirstName string              `json:"first_name"`
	LastName  string              `json:"last_name"`
	Username  string              `json:"username" gorm:"size:255;unique;not null"`
	Phone     string              `json:"phone" gorm:"size:255"`
	Email     string              `json:"email" gorm:"size:255;unique;not null"`
	Password  string              `json:"-" gorm:"size:255;not null"` // Hidden from JSON responses
	RoleId    uint                `json:"role_id"`
	Role      *authorization.Role `json:"role" gorm:"foreignKey:RoleId;references:Id"`
}

// TableName returns the table name for the Employee model
func (m *Employee) TableName() string {
	return "users"
}

// GetId returns the Id of the model
func (m *Employee) GetId() uint {
	return m.Id
}

// GetModelName returns the model name
func (m *Employee) GetModelName() string {
	return "employee"
}

// CreateEmployeeRequest represents the request payload for creating a Employee
type CreateEmployeeRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	RoleId    uint   `json:"role_id"`
}

// UpdateEmployeeRequest represents the request payload for updating a Employee
type UpdateEmployeeRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Email     string `json:"email,omitempty"`
	RoleId    uint   `json:"role_id,omitempty"`
}

// EmployeeResponse represents the API response for Employee
type EmployeeResponse struct {
	Id        uint               `json:"id"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	DeletedAt gorm.DeletedAt     `json:"deleted_at"`
	FirstName string             `json:"first_name"`
	LastName  string             `json:"last_name"`
	Username  string             `json:"username"`
	Phone     string             `json:"phone"`
	Email     string             `json:"email"`
	RoleId    uint               `json:"role_id"`
	Role      authorization.Role `json:"role"`
}

// EmployeeModelResponse represents a simplified response when this model is part of other entities
type EmployeeModelResponse struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // Display name
}

// EmployeeSelectOption represents a simplified response for select boxes and dropdowns
type EmployeeSelectOption struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // Display name
}

// EmployeeListResponse represents the response for list operations (optimized for performance)
type EmployeeListResponse struct {
	Id        uint               `json:"id"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	DeletedAt gorm.DeletedAt     `json:"deleted_at"`
	FirstName string             `json:"first_name"`
	LastName  string             `json:"last_name"`
	Username  string             `json:"username"`
	Phone     string             `json:"phone"`
	Email     string             `json:"email"`
	RoleId    uint               `json:"role_id"`
	Role      authorization.Role `json:"role"`
}

// ToResponse converts the model to an API response
func (m *Employee) ToResponse() *EmployeeResponse {
	if m == nil {
		return nil
	}
	response := &EmployeeResponse{
		Id:        m.Id,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Username:  m.Username,
		Phone:     m.Phone,
		Email:     m.Email,
		RoleId:    m.RoleId,
	}

	// Handle role relationship
	if m.Role != nil {
		response.Role = *m.Role
	}

	return response
}

// ToModelResponse converts the model to a simplified response for when it's part of other entities
func (m *Employee) ToModelResponse() *EmployeeModelResponse {
	if m == nil {
		return nil
	}
	return &EmployeeModelResponse{
		Id:   m.Id,
		Name: fmt.Sprintf("Employee #%d", m.Id), // Fallback to ID-based display
	}
}

// ToSelectOption converts the model to a select option for dropdowns
func (m *Employee) ToSelectOption() *EmployeeSelectOption {
	if m == nil {
		return nil
	}

	// Build display name with proper fallbacks
	var displayName string
	if m.FirstName != "" && m.LastName != "" {
		displayName = m.FirstName + " " + m.LastName
	} else if m.FirstName != "" {
		displayName = m.FirstName
	} else if m.LastName != "" {
		displayName = m.LastName
	} else if m.Username != "" {
		displayName = m.Username
	} else if m.Email != "" {
		displayName = m.Email
	} else {
		displayName = fmt.Sprintf("Employee #%d", m.Id)
	}

	return &EmployeeSelectOption{
		Id:   m.Id,
		Name: displayName,
	}
}

// ToListResponse converts the model to a list response (without preloaded relationships for fast listing)
func (m *Employee) ToListResponse() *EmployeeListResponse {
	if m == nil {
		return nil
	}
	response := &EmployeeListResponse{
		Id:        m.Id,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Username:  m.Username,
		Phone:     m.Phone,
		Email:     m.Email,
		RoleId:    m.RoleId,
	}

	// Handle role relationship
	if m.Role != nil {
		response.Role = *m.Role
	}

	return response
}

// Preload preloads all the model's relationships
func (m *Employee) Preload(db *gorm.DB) *gorm.DB {
	query := db.Preload("Role")
	return query
}
