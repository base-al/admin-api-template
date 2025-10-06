package notifications

import (
	"base/app/models"
	"base/core/app/authorization"
	"base/core/module"
	"base/core/router"
	"errors"

	"gorm.io/gorm"
)

type Module struct {
	module.DefaultModule
	DB         *gorm.DB
	Service    *NotificationService
	Controller *NotificationController
}

// Init creates and initializes the Notification module with all dependencies
func Init(deps module.Dependencies) module.Module {
	// Initialize service and controller
	service := NewNotificationService(deps.DB, deps.Emitter, deps.Storage, deps.Logger)
	controller := NewNotificationController(service, deps.Storage)

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
	// Auto-migrate the model
	if err := m.Migrate(); err != nil {
		return err
	}

	return m.SeedPermissions()
}

func (m *Module) SeedPermissions() error {
	// Ensure permissions table exists before seeding
	if err := m.DB.AutoMigrate(&authorization.Permission{}); err != nil {
		return err
	}

	// Define permissions for notification CRUD operations
	notificationPermissions := []authorization.Permission{
		{
			Name:         "notification list",
			Description:  "View notification list",
			ResourceType: "notification",
			Action:       "list",
		},
		{
			Name:         "notification read",
			Description:  "View notification details",
			ResourceType: "notification",
			Action:       "read",
		},
		{
			Name:         "notification create",
			Description:  "Create new notifications",
			ResourceType: "notification",
			Action:       "create",
		},
		{
			Name:         "notification update",
			Description:  "Update notification information",
			ResourceType: "notification",
			Action:       "update",
		},
		{
			Name:         "notification delete",
			Description:  "Delete notifications",
			ResourceType: "notification",
			Action:       "delete",
		},
	}

	// Upsert permissions - create or update if they exist
	for _, permission := range notificationPermissions {
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

func (m *Module) Migrate() error {
	return m.DB.AutoMigrate(&models.Notification{})
}

func (m *Module) GetModels() []any {
	return []any{
		&models.Notification{},
	}
}
