package employees

import (
	"errors"
	"math"

	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"
	"base/core/types"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)


const (
	CreateEmployeeEvent = "employees.create"
	UpdateEmployeeEvent = "employees.update"
	DeleteEmployeeEvent = "employees.delete"
)

type EmployeeService struct {
	DB      *gorm.DB
	Emitter *emitter.Emitter
	Storage *storage.ActiveStorage
	Logger  logger.Logger
}

func NewEmployeeService(db *gorm.DB, emitter *emitter.Emitter, storage *storage.ActiveStorage, logger logger.Logger) *EmployeeService {
	return &EmployeeService{
		DB:      db,
		Logger:  logger,
		Emitter: emitter,
		Storage: storage,
	}
}

// applySorting applies sorting to the query based on the sort and order parameters
func (s *EmployeeService) applySorting(query *gorm.DB, sortBy *string, sortOrder *string) {
	// Valid sortable fields for Employee
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
		"password":   "password",
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

func (s *EmployeeService) Create(req *CreateEmployeeRequest) (*Employee, error) {
	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Error("failed to hash password", logger.String("error", err.Error()))
		return nil, err
	}

	item := &Employee{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Phone:     req.Phone,
		Email:     req.Email,
		Password:  string(hashedPassword),
		RoleId:    req.RoleId,
	}

	if err := s.DB.Create(item).Error; err != nil {
		s.Logger.Error("failed to create employee", logger.String("error", err.Error()))
		return nil, err
	}

	// Emit create event
	s.Emitter.Emit(CreateEmployeeEvent, item)

	return s.GetById(item.Id)
}

func (s *EmployeeService) Update(id uint, req *UpdateEmployeeRequest) (*Employee, error) {
	item := &Employee{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find employee for update",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Validate request
	if err := ValidateEmployeeUpdateRequest(req, id); err != nil {
		return nil, err
	}

	// Update fields directly on the model
	// For non-pointer string fields
	if req.FirstName != "" {
		item.FirstName = req.FirstName
	}
	// For non-pointer string fields
	if req.LastName != "" {
		item.LastName = req.LastName
	}
	// For non-pointer string fields
	if req.Username != "" {
		item.Username = req.Username
	}
	// For non-pointer string fields
	if req.Phone != "" {
		item.Phone = req.Phone
	}
	// For non-pointer string fields
	if req.Email != "" {
		item.Email = req.Email
	}
	// For non-pointer unsigned integer fields
	if req.RoleId != 0 {
		item.RoleId = req.RoleId
	}

	if err := s.DB.Save(item).Error; err != nil {
		s.Logger.Error("failed to update employee",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Handle many-to-many relationships

	result, err := s.GetById(item.Id)
	if err != nil {
		s.Logger.Error("failed to get updated employee",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	// Emit update event
	s.Emitter.Emit(UpdateEmployeeEvent, result)

	return result, nil
}

func (s *EmployeeService) Delete(id uint) error {
	item := &Employee{}
	if err := s.DB.First(item, id).Error; err != nil {
		s.Logger.Error("failed to find employee for deletion",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Delete file attachments if any

	if err := s.DB.Delete(item).Error; err != nil {
		s.Logger.Error("failed to delete employee",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Emit delete event
	s.Emitter.Emit(DeleteEmployeeEvent, item)

	return nil
}

func (s *EmployeeService) GetById(id uint) (*Employee, error) {
	item := &Employee{}

	query := item.Preload(s.DB)
	if err := query.First(item, id).Error; err != nil {
		s.Logger.Error("failed to get employee",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return nil, err
	}

	return item, nil
}

func (s *EmployeeService) GetAll(page *int, limit *int, sortBy *string, sortOrder *string) (*types.PaginatedResponse, error) {
	var items []*Employee
	var total int64

	query := s.DB.Model(&Employee{})
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
		s.Logger.Error("failed to count employees",
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

	// Preload relationships for list response
	query = (&Employee{}).Preload(query)

	// Execute query
	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("failed to get employees",
			logger.String("error", err.Error()))
		return nil, err
	}

	// Convert to response type
	responses := make([]*EmployeeListResponse, len(items))
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
func (s *EmployeeService) GetAllForSelect() ([]*Employee, error) {
	var items []*Employee

	query := s.DB.Model(&Employee{})

	// Only select the necessary fields for select options
	query = query.Select("id, first_name, last_name, username, email") // Select fields needed for display name

	// Order by name/title for better UX
	query = query.Order("id ASC")

	if err := query.Find(&items).Error; err != nil {
		s.Logger.Error("Failed to fetch items for select", logger.String("error", err.Error()))
		return nil, err
	}

	return items, nil
}

// ChangeEmployeePassword changes the password for a specific employee
func (s *EmployeeService) ChangeEmployeePassword(id uint, newPassword, currentPassword string) error {
	var item Employee
	if err := s.DB.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.Logger.Error("Employee not found for password change",
				logger.Int("id", int(id)))
		} else {
			s.Logger.Error("Database error while fetching employee for password change",
				logger.String("error", err.Error()),
				logger.Int("id", int(id)))
		}
		return err
	}

	// Verify current password using bcrypt (skip if currentPassword is empty - admin override)
	if currentPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(item.Password), []byte(currentPassword)); err != nil {
			s.Logger.Info("Invalid current password provided for employee",
				logger.Int("id", int(id)))
			return bcrypt.ErrMismatchedHashAndPassword
		}
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Error("Failed to hash new password for employee",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	// Set the hashed password
	item.Password = string(hashedPassword)

	if err := s.DB.Save(&item).Error; err != nil {
		s.Logger.Error("Failed to save new password for employee",
			logger.String("error", err.Error()),
			logger.Int("id", int(id)))
		return err
	}

	s.Logger.Info("Employee password changed successfully", logger.Int("id", int(id)))
	return nil
}

// GetEmployeeTasks gets all tasks assigned to a specific employee
// This is a placeholder for future task management functionality
func (s *EmployeeService) GetEmployeeTasks(employeeId uint) ([]map[string]interface{}, error) {
	s.Logger.Info("Getting employee tasks", logger.Int("employee_id", int(employeeId)))

	// Return empty array for now - can be extended with task management module
	return []map[string]interface{}{}, nil
}
