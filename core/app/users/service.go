package users

import (
	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"
	"base/core/types"
	"context"
	"errors"
	"fmt"
	"math"
	"mime/multipart"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	CreateUserEvent = "users.create"
	UpdateUserEvent = "users.update"
	DeleteUserEvent = "users.delete"
)

type UserService struct {
	db            *gorm.DB
	emitter       *emitter.Emitter
	activeStorage *storage.ActiveStorage
	logger        logger.Logger
}

func NewUserService(db *gorm.DB, emitter *emitter.Emitter, activeStorage *storage.ActiveStorage, logger logger.Logger) *UserService {
	if db == nil {
		panic("db is required")
	}
	if logger == nil {
		panic("logger is required")
	}
	if activeStorage == nil {
		panic("activeStorage is required")
	}

	// Register avatar attachment configuration
	activeStorage.RegisterAttachment("users", storage.AttachmentConfig{
		Field:             "avatar",
		Path:              "avatars",
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif"},
		MaxFileSize:       5 << 20, // 5MB
		Multiple:          false,
	})

	return &UserService{
		db:            db,
		emitter:       emitter,
		activeStorage: activeStorage,
		logger:        logger,
	}
}

// applySorting applies sorting to the query based on the sort and order parameters
func (s *UserService) applySorting(query *gorm.DB, sortBy *string, sortOrder *string) {
	validSortFields := map[string]string{
		"id":         "id",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"first_name": "first_name",
		"last_name":  "last_name",
		"username":   "username",
		"phone":      "phone",
		"email":      "email",
		"role_id":    "role_id",
	}

	defaultSortBy := "id"
	defaultSortOrder := "desc"

	sortField := defaultSortBy
	if sortBy != nil && *sortBy != "" {
		if field, exists := validSortFields[*sortBy]; exists {
			sortField = field
		}
	}

	sortDirection := defaultSortOrder
	if sortOrder != nil && (*sortOrder == "asc" || *sortOrder == "desc") {
		sortDirection = *sortOrder
	}

	query.Order(sortField + " " + sortDirection)
}

// Create creates a new user
func (s *UserService) Create(req *CreateUserRequest) (*User, error) {
	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", logger.String("error", err.Error()))
		return nil, err
	}

	item := &User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Phone:     req.Phone,
		Email:     req.Email,
		Password:  string(hashedPassword),
		RoleId:    req.RoleId,
	}

	if err := s.db.Create(item).Error; err != nil {
		s.logger.Error("failed to create user", logger.String("error", err.Error()))
		return nil, err
	}

	// Emit create event
	s.emitter.Emit(CreateUserEvent, item)

	return s.GetById(item.Id)
}

// GetById gets a user by ID with relationships preloaded
func (s *UserService) GetById(id uint) (*User, error) {
	var user User
	query := user.Preload(s.db)
	if err := query.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("User not found", logger.Uint("user_id", id))
		} else {
			s.logger.Error("Database error while fetching user", logger.Uint("user_id", id))
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (s *UserService) Update(id uint, req *UpdateUserRequest) (*User, error) {
	item := &User{}
	if err := s.db.First(item, id).Error; err != nil {
		s.logger.Error("failed to find user for update",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Update fields if provided
	if req.FirstName != "" {
		item.FirstName = req.FirstName
	}
	if req.LastName != "" {
		item.LastName = req.LastName
	}
	if req.Username != "" {
		item.Username = req.Username
	}
	if req.Phone != "" {
		item.Phone = req.Phone
	}
	if req.Email != "" {
		item.Email = req.Email
	}
	if req.RoleId != 0 {
		item.RoleId = req.RoleId
	}

	if err := s.db.Save(item).Error; err != nil {
		s.logger.Error("failed to update user",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	result, err := s.GetById(item.Id)
	if err != nil {
		s.logger.Error("failed to get updated user",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Emit update event
	s.emitter.Emit(UpdateUserEvent, result)

	return result, nil
}

// Delete deletes a user
func (s *UserService) Delete(id uint) error {
	item := &User{}
	if err := s.db.First(item, id).Error; err != nil {
		s.logger.Error("failed to find user for deletion",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Delete avatar if exists
	if item.Avatar != nil {
		if err := s.activeStorage.Delete(item.Avatar); err != nil {
			s.logger.Error("Failed to delete avatar",
				logger.String("error", err.Error()),
				logger.Uint("user_id", id))
		}
	}

	if err := s.db.Delete(item).Error; err != nil {
		s.logger.Error("failed to delete user",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Emit delete event
	s.emitter.Emit(DeleteUserEvent, item)

	return nil
}

// GetAll gets all users with pagination
func (s *UserService) GetAll(page *int, limit *int, sortBy *string, sortOrder *string) (*types.PaginatedResponse, error) {
	var items []*User
	var total int64

	query := s.db.Model(&User{})

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
		s.logger.Error("failed to count users", logger.String("error", err.Error()))
		return nil, err
	}

	// Apply pagination
	if page != nil && limit != nil {
		offset := (*page - 1) * *limit
		query = query.Offset(offset).Limit(*limit)
	}

	// Apply sorting
	s.applySorting(query, sortBy, sortOrder)

	// Preload relationships
	query = (&User{}).Preload(query)

	// Execute query
	if err := query.Find(&items).Error; err != nil {
		s.logger.Error("failed to get users", logger.String("error", err.Error()))
		return nil, err
	}

	// Convert to response type
	responses := make([]*UserResponse, len(items))
	for i, item := range items {
		responses[i] = item.ToResponse()
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

// GetAllForSelect gets all users for select box/dropdown options
func (s *UserService) GetAllForSelect() ([]*User, error) {
	var items []*User

	query := s.db.Model(&User{})
	query = query.Select("id, first_name, last_name, username, email")
	query = query.Order("id ASC")

	if err := query.Find(&items).Error; err != nil {
		s.logger.Error("Failed to fetch users for select", logger.String("error", err.Error()))
		return nil, err
	}

	return items, nil
}

// UpdateAvatar updates user's avatar
func (s *UserService) UpdateAvatar(ctx context.Context, id uint, avatarFile *multipart.FileHeader) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}

	// Attach the new file - cleanup is handled inside Attach
	attachment, err := s.activeStorage.Attach(&user, "avatar", avatarFile)
	if err != nil {
		return nil, fmt.Errorf("failed to upload avatar: %w", err)
	}

	// Update user's avatar
	user.Avatar = attachment
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

// RemoveAvatar removes user's avatar
func (s *UserService) RemoveAvatar(ctx context.Context, id uint) (*User, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var user User
	if err := tx.First(&user, id).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if user.Avatar != nil {
		if err := s.activeStorage.Delete(user.Avatar); err != nil {
			tx.Rollback()
			s.logger.Error("Failed to delete avatar",
				logger.String("error", err.Error()),
				logger.Uint("user_id", id))
			return nil, fmt.Errorf("failed to delete avatar: %w", err)
		}
		user.Avatar = nil
		if err := tx.Save(&user).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdatePassword updates own password (requires old password verification)
func (s *UserService) UpdatePassword(id uint, req *UpdatePasswordRequest) error {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		s.logger.Error("Failed to find user for password update",
			logger.String("error", err.Error()),
			logger.Uint("user_id", id))
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		s.logger.Info("Invalid old password provided", logger.Uint("user_id", id))
		return bcrypt.ErrMismatchedHashAndPassword
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash new password",
			logger.String("error", err.Error()),
			logger.Uint("user_id", id))
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.Password = string(hashedPassword)
	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to save new password",
			logger.String("error", err.Error()),
			logger.Uint("user_id", id))
		return fmt.Errorf("failed to update user password: %w", err)
	}

	return nil
}

// ChangePassword changes password for another user (admin only - no old password verification)
func (s *UserService) ChangePassword(id uint, newPassword, currentPassword string) error {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Error("User not found for password change", logger.Int("id", int(id)))
		} else {
			s.logger.Error("Database error while fetching user for password change",
				logger.String("error", err.Error()),
				logger.Int("id", int(id)))
		}
		return err
	}

	// Verify current password using bcrypt (skip if currentPassword is empty - admin override)
	if currentPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
			s.logger.Info("Invalid current password provided for user", logger.Int("id", int(id)))
			return bcrypt.ErrMismatchedHashAndPassword
		}
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash new password for user",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	user.Password = string(hashedPassword)

	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to save new password for user",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	s.logger.Info("User password changed successfully", logger.Int("id", int(id)))
	return nil
}

// GetUserTasks gets all tasks assigned to a specific user
// This is a placeholder for future task management functionality
func (s *UserService) GetUserTasks(userId uint) ([]map[string]interface{}, error) {
	s.logger.Info("Getting user tasks", logger.Int("user_id", int(userId)))

	// Return empty array for now - can be extended with task management module
	return []map[string]interface{}{}, nil
}
