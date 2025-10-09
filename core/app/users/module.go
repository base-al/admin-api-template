package users

import (
	"errors"

	"base/core/app/authorization"
	"base/core/module"
	"base/core/router"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Module struct {
	module.DefaultModule
	DB         *gorm.DB
	Service    *UserService
	Controller *UserController
}

// Init creates and initializes the User module with all dependencies
func Init(deps module.Dependencies) module.Module {
	// Initialize service and controller
	service := NewUserService(deps.DB, deps.Emitter, deps.Storage, deps.Logger)
	controller := NewUserController(service, deps.Storage, deps.Logger)

	// Create module
	mod := &Module{
		DB:         deps.DB,
		Service:    service,
		Controller: controller,
	}

	return mod
}

// Routes registers the module routes
func (m *Module) Routes(router *router.RouterGroup) {
	m.Controller.Routes(router)
}

func (m *Module) Init() error {
	return nil
}

func (m *Module) Migrate() error {
	err := m.DB.AutoMigrate(&User{})
	if err != nil {
		return err
	}

	if err := m.SeedPermissions(); err != nil {
		return err
	}

	return m.SeedDefaultUser()
}

func (m *Module) SeedPermissions() error {
	// Ensure permissions table exists before seeding
	if err := m.DB.AutoMigrate(&authorization.Permission{}); err != nil {
		return err
	}

	// Define permissions for user management endpoints (admin only)
	userPermissions := []authorization.Permission{
		{
			Name:         "user list",
			Description:  "View user list (paginated)",
			ResourceType: "user",
			Action:       "list",
		},
		{
			Name:         "user list_all",
			Description:  "View all users (unpaginated)",
			ResourceType: "user",
			Action:       "list_all",
		},
		{
			Name:         "user read",
			Description:  "View user details",
			ResourceType: "user",
			Action:       "read",
		},
		{
			Name:         "user create",
			Description:  "Create new users",
			ResourceType: "user",
			Action:       "create",
		},
		{
			Name:         "user update",
			Description:  "Update user information",
			ResourceType: "user",
			Action:       "update",
		},
		{
			Name:         "user delete",
			Description:  "Delete users",
			ResourceType: "user",
			Action:       "delete",
		},
	}

	// Upsert permissions - create or update if they exist
	for _, permission := range userPermissions {
		var existingPermission authorization.Permission
		result := m.DB.Where("resource_type = ? AND action = ?", permission.ResourceType, permission.Action).First(&existingPermission)

		if result.Error != nil && errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new permission
			if err := m.DB.Create(&permission).Error; err != nil {
				return err
			}
		} else if result.Error == nil {
			// Update existing permission
			existingPermission.Name = permission.Name
			existingPermission.Description = permission.Description
			if err := m.DB.Save(&existingPermission).Error; err != nil {
				return err
			}
		} else {
			// Return any other error
			return result.Error
		}
	}

	return nil
}

func (m *Module) SeedDefaultUser() error {
	// Check if any users exist
	var count int64
	if err := m.DB.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}

	// If users already exist, skip seeding
	if count > 0 {
		return nil
	}

	// Get Super Admin role
	var superAdminRole authorization.Role
	if err := m.DB.Where("name = ? AND is_system = ?", "Super Admin", true).First(&superAdminRole).Error; err != nil {
		return err
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create default admin user
	defaultUser := User{
		FirstName: "Super",
		LastName:  "Admin",
		Username:  "admin",
		Email:     "admin@admin.com",
		Password:  string(hashedPassword),
		RoleId:    superAdminRole.Id,
	}

	return m.DB.Create(&defaultUser).Error
}

func (m *Module) GetModels() []any {
	return []any{
		&User{},
	}
}
