package users

import (
	"base/core/app/authorization"
	"base/core/storage"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// User represents a user entity (used for both profile and employee management)
type User struct {
	Id        uint                `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	FirstName string              `json:"first_name" gorm:"column:first_name;not null;size:255"`
	LastName  string              `json:"last_name" gorm:"column:last_name;not null;size:255"`
	Username  string              `json:"username" gorm:"column:username;unique;not null;size:255"`
	Phone     string              `json:"phone" gorm:"column:phone;size:255"`
	Email     string              `json:"email" gorm:"column:email;unique;not null;size:255"`
	Password  string              `json:"-" gorm:"column:password;size:255;not null"` // Hidden from JSON
	RoleId    uint                `json:"role_id" gorm:"column:role_id;default:3"`
	Role      *authorization.Role `json:"role,omitempty" gorm:"foreignKey:RoleId;references:Id"`
	Avatar    *storage.Attachment `json:"avatar,omitempty" gorm:"foreignKey:ModelId;references:Id"`
	LastLogin *time.Time          `json:"last_login,omitempty" gorm:"column:last_login"`
	CreatedAt time.Time           `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time           `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt      `json:"deleted_at,omitempty" gorm:"column:deleted_at;index"`
}

// TableName returns the table name for the User model
func (m *User) TableName() string {
	return "users"
}

// GetId returns the Id of the model (for storage attachments)
func (m *User) GetId() uint {
	return m.Id
}

// GetModelName returns the model name (for storage attachments)
func (m *User) GetModelName() string {
	return "users"
}

// CreateUserRequest represents the request payload for creating a User
type CreateUserRequest struct {
	FirstName string `json:"first_name" binding:"required,max=255"`
	LastName  string `json:"last_name" binding:"required,max=255"`
	Username  string `json:"username" binding:"required,max=255"`
	Phone     string `json:"phone" binding:"max=255"`
	Email     string `json:"email" binding:"required,email,max=255"`
	Password  string `json:"password" binding:"required,min=8,max=255"`
	RoleId    uint   `json:"role_id"`
}

// UpdateUserRequest represents the request payload for updating a User
type UpdateUserRequest struct {
	FirstName string `json:"first_name,omitempty" binding:"max=255"`
	LastName  string `json:"last_name,omitempty" binding:"max=255"`
	Username  string `json:"username,omitempty" binding:"max=255"`
	Phone     string `json:"phone,omitempty" binding:"max=255"`
	Email     string `json:"email,omitempty" binding:"email,max=255"`
	RoleId    uint   `json:"role_id,omitempty"`
}

// UpdatePasswordRequest represents the request for updating own password
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,max=255"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=255"`
}

// ChangePasswordRequest represents the request for changing another user's password (admin only)
type ChangePasswordRequest struct {
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	CurrentPassword string `json:"current_password" binding:"required"`
}

// UserResponse represents the API response for User
type UserResponse struct {
	Id        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	RoleId    uint   `json:"role_id"`
	RoleName  string `json:"role_name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	LastLogin string `json:"last_login,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// UserSelectOption represents a simplified response for select boxes and dropdowns
type UserSelectOption struct {
	Id   uint   `json:"id"`
	Name string `json:"name"`
}

// UserModelResponse represents a simplified response when User is part of other entities
type UserModelResponse struct {
	Id        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
}

// ToResponse converts the User to a UserResponse
func (m *User) ToResponse() *UserResponse {
	if m == nil {
		return nil
	}
	response := &UserResponse{
		Id:        m.Id,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Username:  m.Username,
		Phone:     m.Phone,
		Email:     m.Email,
		RoleId:    m.RoleId,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		UpdatedAt: m.UpdatedAt.Format(time.RFC3339),
	}

	// Include role name if role relationship is loaded
	if m.Role != nil {
		response.RoleName = m.Role.Name
	}

	// Include avatar URL if avatar is loaded
	if m.Avatar != nil {
		response.AvatarURL = m.Avatar.URL
	}

	// Include last login if available
	if m.LastLogin != nil {
		response.LastLogin = m.LastLogin.Format(time.RFC3339)
	}

	return response
}

// ToSelectOption converts the model to a select option for dropdowns
func (m *User) ToSelectOption() *UserSelectOption {
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
		displayName = fmt.Sprintf("User #%d", m.Id)
	}

	return &UserSelectOption{
		Id:   m.Id,
		Name: displayName,
	}
}

// ToModelResponse converts the model to a simplified response for when it's part of other entities
func (m *User) ToModelResponse() *UserModelResponse {
	if m == nil {
		return nil
	}
	return &UserModelResponse{
		Id:        m.Id,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		Username:  m.Username,
		Email:     m.Email,
	}
}

// Preload preloads all the model's relationships
func (m *User) Preload(db *gorm.DB) *gorm.DB {
	query := db.Preload("Role").Preload("Avatar")
	return query
}
