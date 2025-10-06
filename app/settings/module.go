package settings

import (
	"errors"

	"base/app/models"
	"base/core/app/authorization"
	"base/core/module"
	"base/core/router"

	"gorm.io/gorm"
)

type Module struct {
	module.DefaultModule
	DB         *gorm.DB
	Service    *SettingsService
	Controller *SettingsController
}

// Init creates and initializes the Settings module with all dependencies
func Init(deps module.Dependencies) module.Module {
	// Initialize service and controller
	service := NewSettingsService(deps.DB, deps.Emitter, deps.Storage, deps.Logger)
	controller := NewSettingsController(service, deps.Storage)

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
	// Run auto migration first
	if err := m.DB.AutoMigrate(&models.Settings{}); err != nil {
		return err
	}

	// Seed default settings
	if err := m.seedDefaultSettings(); err != nil {
		return err
	}

	// Seed permissions
	return m.SeedPermissions()
}

// seedDefaultSettings creates default system settings if they don't exist
func (m *Module) seedDefaultSettings() error {
	defaultSettings := []models.Settings{
		// Company Information
		{
			SettingKey:  "company_name",
			Label:       "Company Name",
			Group:       "company",
			Type:        "string",
			ValueString: "Your Company",
			Description: "Company name displayed on documents",
			IsPublic:    true,
		},
		{
			SettingKey:  "company_address",
			Label:       "Company Address",
			Group:       "company",
			Type:        "string",
			ValueString: "123 Main Street, City, Country",
			Description: "Company business address",
			IsPublic:    true,
		},
		{
			SettingKey:  "company_phone",
			Label:       "Company Phone",
			Group:       "company",
			Type:        "string",
			ValueString: "+1 234 567 890",
			Description: "Company contact phone number",
			IsPublic:    true,
		},
		{
			SettingKey:  "company_email",
			Label:       "Company Email",
			Group:       "company",
			Type:        "string",
			ValueString: "info@yourcompany.com",
			Description: "Company contact email address",
			IsPublic:    true,
		},
		{
			SettingKey:  "company_nui",
			Label:       "Tax Number",
			Group:       "company",
			Type:        "string",
			ValueString: "",
			Description: "Company tax identification number",
			IsPublic:    false,
		},
		{
			SettingKey:  "company_website",
			Label:       "Company Website",
			Group:       "company",
			Type:        "string",
			ValueString: "https://yourcompany.com",
			Description: "Company website URL",
			IsPublic:    true,
		},

		// Email Settings
		{
			SettingKey:  "email_from_name",
			Label:       "Email From Name",
			Group:       "email",
			Type:        "string",
			ValueString: "Your Company Support",
			Description: "Default sender name for system emails",
			IsPublic:    false,
		},
		{
			SettingKey:  "email_signature",
			Label:       "Email Signature",
			Group:       "email",
			Type:        "string",
			ValueString: "Best regards,\nYour Company Team\ninfo@yourcompany.com",
			Description: "Default email signature",
			IsPublic:    false,
		},

		// System Settings
		{
			SettingKey:  "maintenance_mode",
			Label:       "Maintenance Mode",
			Group:       "system",
			Type:        "bool",
			ValueBool:   false,
			Description: "Enable maintenance mode for the system",
			IsPublic:    true,
		},
		{
			SettingKey:  "timezone",
			Label:       "System Timezone",
			Group:       "system",
			Type:        "string",
			ValueString: "UTC",
			Description: "Default system timezone",
			IsPublic:    false,
		},
		{
			SettingKey:  "date_format",
			Label:       "Date Format",
			Group:       "system",
			Type:        "string",
			ValueString: "YYYY-MM-DD",
			Description: "Default date format for the system",
			IsPublic:    false,
		},
		{
			SettingKey:  "time_format",
			Label:       "Time Format",
			Group:       "system",
			Type:        "string",
			ValueString: "24h",
			Description: "Time format (12h or 24h)",
			IsPublic:    false,
		},

		// Security Settings
		{
			SettingKey:  "session_timeout",
			Label:       "Session Timeout",
			Group:       "security",
			Type:        "int",
			ValueInt:    3600,
			Description: "User session timeout in seconds",
			IsPublic:    false,
		},
		{
			SettingKey:  "password_min_length",
			Label:       "Minimum Password Length",
			Group:       "security",
			Type:        "int",
			ValueInt:    8,
			Description: "Minimum required password length",
			IsPublic:    false,
		},
		{
			SettingKey:  "enable_2fa",
			Label:       "Enable Two-Factor Authentication",
			Group:       "security",
			Type:        "bool",
			ValueBool:   false,
			Description: "Enable two-factor authentication for users",
			IsPublic:    false,
		},
	}

	// Insert settings that don't already exist
	for _, setting := range defaultSettings {
		var existing models.Settings
		result := m.DB.Where("setting_key = ?", setting.SettingKey).First(&existing)
		
		// If setting doesn't exist, create it
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if err := m.DB.Create(&setting).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Module) SeedPermissions() error {
	// Ensure permissions table exists before seeding
	if err := m.DB.AutoMigrate(&authorization.Permission{}); err != nil {
		return err
	}

	// Define permissions based on actual controller endpoints:
	// GET /settings (List), POST /settings (Create), GET /settings/all (ListAll)
	// GET /settings/:id (Get), PUT /settings/:id (Update), DELETE /settings/:id (Delete)

	settingsPermissions := []authorization.Permission{
		{
			Name:         "settings list",
			Description:  "View settings list (paginated)",
			ResourceType: "settings",
			Action:       "list",
		},
		{
			Name:         "settings list_all",
			Description:  "View all settings (unpaginated)",
			ResourceType: "settings",
			Action:       "list_all",
		},
		{
			Name:         "settings read",
			Description:  "View settings details",
			ResourceType: "settings",
			Action:       "read",
		},
		{
			Name:         "settings create",
			Description:  "Create new settings",
			ResourceType: "settings",
			Action:       "create",
		},
		{
			Name:         "settings update",
			Description:  "Update settings information",
			ResourceType: "settings",
			Action:       "update",
		},
		{
			Name:         "settings delete",
			Description:  "Delete settings",
			ResourceType: "settings",
			Action:       "delete",
		},
	}

	// Upsert permissions - create or update if they exist
	for _, permission := range settingsPermissions {
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

	// Assign all settings permissions to Super Admin role
	var superAdminRole authorization.Role
	if err := m.DB.Where("name = ? AND is_system = ?", "Super Admin", true).First(&superAdminRole).Error; err == nil {
		// Get all settings permissions
		var settingsPerms []authorization.Permission
		if err := m.DB.Where("resource_type = ?", "settings").Find(&settingsPerms).Error; err != nil {
			return err
		}

		for _, permission := range settingsPerms {
			var rolePermission authorization.RolePermission
			result := m.DB.Where("role_id = ? AND permission_id = ?", superAdminRole.Id, permission.Id).First(&rolePermission)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				rolePermission = authorization.RolePermission{
					RoleId:       superAdminRole.Id,
					PermissionId: permission.Id,
				}
				if err := m.DB.Create(&rolePermission).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m *Module) GetModels() []any {
	return []any{
		&models.Settings{},
	}
}
