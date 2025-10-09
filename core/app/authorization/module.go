package authorization

import (
	"errors"

	"base/core/logger"
	"base/core/module"
	"base/core/router"
	"strings"

	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type AuthorizationModule struct {
	module.DefaultModule
	DB         *gorm.DB
	Controller *AuthorizationController
	Service    *AuthorizationService
	Logger     logger.Logger
}

func NewAuthorizationModule(db *gorm.DB, router *router.RouterGroup, logger logger.Logger) module.Module {
	service := NewAuthorizationService(db)
	controller := NewAuthorizationController(service, logger)

	authzModule := &AuthorizationModule{
		DB:         db,
		Controller: controller,
		Service:    service,
		Logger:     logger,
	}

	return authzModule
}

func (m *AuthorizationModule) Routes(router *router.RouterGroup) {
	// Router is already within api group from start.go
	m.Logger.Info("Registering authorization module routes")
	m.Controller.Routes(router)
	m.Logger.Info("Authorization module routes registered successfully")
}

func (m *AuthorizationModule) Migrate() error {
	err := m.DB.AutoMigrate(
		&Role{},
		&Permission{},
		&RolePermission{},
		&ResourcePermission{},
		&ResourceAccess{},
	)
	if err != nil {
		return err
	}

	// Seed default roles and permissions
	if err := m.seedDefaultData(); err != nil {
		m.Logger.Error("Failed to seed authorization data", logger.String("error", err.Error()))
		return err
	}

	return nil
}

func (m *AuthorizationModule) GetObject(foreignKey string, dbTableName string) []any {

	var result []any
	query := m.DB.Table(dbTableName).Where(foreignKey)
	query.Find(&result)
	return result
}

// seedDefaultData creates default roles and permissions if they don't exist
func (m *AuthorizationModule) seedDefaultData() error {
	// Define default system roles for starter kit
	defaultRoles := []Role{
		{
			Name:        "Super Admin",
			Description: "Full system access with all permissions",
			IsSystem:    true,
		},
		{
			Name:        "Administrator",
			Description: "System administration and user management",
			IsSystem:    true,
		},
		{
			Name:        "Manager",
			Description: "Team management and oversight",
			IsSystem:    true,
		},
		{
			Name:        "Employee",
			Description: "Standard employee access",
			IsSystem:    true,
		},
		{
			Name:        "Viewer",
			Description: "Read-only access",
			IsSystem:    true,
		},
	}

	// Define core resource types
	resourceTypes := []string{
		"user",
		"authorization",
		"role",
		"permission",
		"media",
		"profile",
		"employee",
		"settings",
		"post",
		"notification",
		"activity",
	}

	// Define standard CRUD actions
	actions := []string{
		"create",
		"read",
		"update",
		"delete",
		"list",
		"list_all",
		"activate",
		"deactivate",
	}

	// Create default permissions based on resources and actions
	var defaultPermissions []Permission
	for _, resourceType := range resourceTypes {
		for _, action := range actions {
			defaultPermissions = append(defaultPermissions, Permission{
				Name:         resourceType + " " + action,
				Description:  "Allows " + action + " operations on " + resourceType,
				ResourceType: resourceType,
				Action:       action,
			})
		}
	}

	// Add special permissions
	specialPermissions := []Permission{
		{
			Name:         "Manage Roles",
			Description:  "Create, update, and delete roles",
			ResourceType: "role",
			Action:       "manage",
		},
		{
			Name:         "Assign Permissions",
			Description:  "Assign permissions to roles",
			ResourceType: "permission",
			Action:       "assign",
		},
	}
	defaultPermissions = append(defaultPermissions, specialPermissions...)

	// Start transaction with silent logger for seeding (to avoid "record not found" noise)
	tx := m.DB.Session(&gorm.Session{Logger: gormLogger.Discard}).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Seed roles
	for _, role := range defaultRoles {
		var existingRole Role
		result := tx.Where("name = ? AND is_system = ?", role.Name, role.IsSystem).First(&existingRole)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if err := tx.Create(&role).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Seed permissions
	for _, permission := range defaultPermissions {
		var existingPermission Permission
		result := tx.Where("resource_type = ? AND action = ?", permission.ResourceType, permission.Action).First(&existingPermission)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if err := tx.Create(&permission).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Assign all permissions to Super Admin role
	var superAdminRole Role
	if err := tx.Where("name = ? AND is_system = ?", "Super Admin", true).First(&superAdminRole).Error; err == nil {
		// Get all permissions
		var allPermissions []Permission
		if err := tx.Find(&allPermissions).Error; err != nil {
			tx.Rollback()
			return err
		}

		for _, permission := range allPermissions {
			var rolePermission RolePermission
			result := tx.Where("role_id = ? AND permission_id = ?", superAdminRole.Id, permission.Id).First(&rolePermission)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				rolePermission = RolePermission{
					RoleId:       superAdminRole.Id,
					PermissionId: permission.Id,
				}
				if err := tx.Create(&rolePermission).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	// Assign appropriate permissions to Admin role
	var adminRole Role
	if err := tx.Where("name = ? AND is_system = ?", "Administrator", true).First(&adminRole).Error; err == nil {
		adminPermissions := []string{
			"user:create", "user:read", "user:update", "user:delete", "user:list", "user:manage_members",
			"authorization:create", "authorization:read", "authorization:update", "authorization:delete", "authorization:list",
			"media:create", "media:read", "media:update", "media:delete", "media:list",
			"profile:create", "profile:read", "profile:update", "profile:delete", "profile:list",
			"role:create", "role:read", "role:update", "role:delete", "role:list",
			"permission:create", "permission:read", "permission:update", "permission:delete", "permission:list",
			"resource_permission:create", "resource_permission:read", "resource_permission:update", "resource_permission:delete", "resource_permission:list",
		}

		for _, permName := range adminPermissions {
			parts := strings.Split(permName, ":")
			if len(parts) != 2 {
				continue
			}
			resourceType, action := parts[0], parts[1]

			var permission Permission
			if err := tx.Where("resource_type = ? AND action = ?", resourceType, action).First(&permission).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue // Skip if permission not found - this is normal
				}
				return err // Only return actual errors
			}

			var rolePermission RolePermission
			result := tx.Where("role_id = ? AND permission_id = ?", adminRole.Id, permission.Id).First(&rolePermission)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				rolePermission = RolePermission{
					RoleId:       adminRole.Id,
					PermissionId: permission.Id,
				}
				if err := tx.Create(&rolePermission).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	// Assign appropriate permissions to Member role
	var memberRole Role
	if err := tx.Where("name = ? AND is_system = ?", "Member", true).First(&memberRole).Error; err == nil {
		memberPermissions := []string{
			"user:read", "user:list",
			"authorization:read", "authorization:list",
			"media:read", "media:list",
			"profile:read", "profile:list",
			"role:read", "role:list",
			"permission:read", "permission:list",
			"resource_permission:read", "resource_permission:list",
		}

		for _, permName := range memberPermissions {
			parts := strings.Split(permName, ":")
			if len(parts) != 2 {
				continue
			}
			resourceType, action := parts[0], parts[1]

			var permission Permission
			if err := tx.Where("resource_type = ? AND action = ?", resourceType, action).First(&permission).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue // Skip if permission not found - this is normal
				}
				return err // Only return actual errors
			}

			var rolePermission RolePermission
			result := tx.Where("role_id = ? AND permission_id = ?", memberRole.Id, permission.Id).First(&rolePermission)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				rolePermission = RolePermission{
					RoleId:       memberRole.Id,
					PermissionId: permission.Id,
				}
				if err := tx.Create(&rolePermission).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	// Assign appropriate permissions to Viewer role
	var viewerRole Role
	if err := tx.Where("name = ? AND is_system = ?", "Viewer", true).First(&viewerRole).Error; err == nil {
		viewerPermissions := []string{
			"user:read", "user:list",
			"authorization:read", "authorization:list",
			"media:read", "media:list",
			"profile:read", "profile:list",
			"role:read", "role:list",
			"permission:read", "permission:list",
			"resource_permission:read", "resource_permission:list",
		}

		for _, permName := range viewerPermissions {
			parts := strings.Split(permName, ":")
			if len(parts) != 2 {
				continue
			}
			resourceType, action := parts[0], parts[1]

			var permission Permission
			if err := tx.Where("resource_type = ? AND action = ?", resourceType, action).First(&permission).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue // Skip if permission not found - this is normal
				}
				return err // Only return actual errors
			}

			var rolePermission RolePermission
			result := tx.Where("role_id = ? AND permission_id = ?", viewerRole.Id, permission.Id).First(&rolePermission)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				rolePermission = RolePermission{
					RoleId:       viewerRole.Id,
					PermissionId: permission.Id,
				}
				if err := tx.Create(&rolePermission).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	// Commit transaction
	return tx.Commit().Error
}

func (m *AuthorizationModule) GetModels() []any {
	return []any{
		&Role{},
		&Permission{},
		&RolePermission{},
		&ResourcePermission{},
		&ResourceAccess{},
	}
}
