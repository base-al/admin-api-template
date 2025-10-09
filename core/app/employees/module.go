package employees

import (
	"errors"

	"base/core/app/authorization"
	"base/core/module"
	"base/core/router"

	"gorm.io/gorm"
)

type Module struct {
	module.DefaultModule
	DB         *gorm.DB
	Service    *EmployeeService
	Controller *EmployeeController
}

// Init creates and initializes the Employee module with all dependencies
func Init(deps module.Dependencies) module.Module {
	// Initialize service and controller
	service := NewEmployeeService(deps.DB, deps.Emitter, deps.Storage, deps.Logger)
	controller := NewEmployeeController(service, deps.Storage)

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
	err := m.DB.AutoMigrate(&Employee{})
	if err != nil {
		return err
	}

	return m.SeedPermissions()
}

func (m *Module) SeedPermissions() error {
	// Ensure permissions table exists before seeding
	if err := m.DB.AutoMigrate(&authorization.Permission{}); err != nil {
		return err
	}

	// Define permissions based on actual controller endpoints:
	// GET /employees (List), POST /employees (Create), GET /employees/all (ListAll)
	// GET /employees/:id (Get), PUT /employees/:id (Update), DELETE /employees/:id (Delete)

	employeePermissions := []authorization.Permission{
		{
			Name:         "employee list",
			Description:  "View employee list (paginated)",
			ResourceType: "employee",
			Action:       "list",
		},
		{
			Name:         "employee list_all",
			Description:  "View all employees (unpaginated)",
			ResourceType: "employee",
			Action:       "list_all",
		},
		{
			Name:         "employee read",
			Description:  "View employee details",
			ResourceType: "employee",
			Action:       "read",
		},
		{
			Name:         "employee create",
			Description:  "Create new employees",
			ResourceType: "employee",
			Action:       "create",
		},
		{
			Name:         "employee update",
			Description:  "Update employee information",
			ResourceType: "employee",
			Action:       "update",
		},
		{
			Name:         "employee delete",
			Description:  "Delete employees",
			ResourceType: "employee",
			Action:       "delete",
		},
	}

	// Upsert permissions - create or update if they exist
	for _, permission := range employeePermissions {
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

func (m *Module) GetModels() []any {
	return []any{
		&Employee{},
	}
}
