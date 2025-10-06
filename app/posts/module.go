package posts

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
	Service    *PostService
	Controller *PostController
}

// Init creates and initializes the Post module with all dependencies
func Init(deps module.Dependencies) module.Module {
	// Initialize service and controller
	service := NewPostService(deps.DB, deps.Emitter, deps.Storage, deps.Logger)
	controller := NewPostController(service, deps.Storage)

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

	// Define permissions for post CRUD operations
	postPermissions := []authorization.Permission{
		{
			Name:         "post list",
			Description:  "View post list",
			ResourceType: "post",
			Action:       "list",
		},
		{
			Name:         "post read",
			Description:  "View post details",
			ResourceType: "post",
			Action:       "read",
		},
		{
			Name:         "post create",
			Description:  "Create new posts",
			ResourceType: "post",
			Action:       "create",
		},
		{
			Name:         "post update",
			Description:  "Update post information",
			ResourceType: "post",
			Action:       "update",
		},
		{
			Name:         "post delete",
			Description:  "Delete posts",
			ResourceType: "post",
			Action:       "delete",
		},
	}

	// Upsert permissions - create or update if they exist
	for _, permission := range postPermissions {
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
	return m.DB.AutoMigrate(&models.Post{})
}

func (m *Module) GetModels() []any {
	return []any{
		&models.Post{},
	}
}
