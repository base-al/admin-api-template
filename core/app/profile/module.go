package profile

import (
	"base/core/logger"
	"base/core/module"
	"base/core/router"
	"base/core/storage"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserModule struct {
	module.DefaultModule
	DB            *gorm.DB
	Controller    *ProfileController
	Service       *ProfileService
	Logger        logger.Logger
	ActiveStorage *storage.ActiveStorage
}

func NewUserModule(
	db *gorm.DB,
	router *router.RouterGroup,
	logger logger.Logger,
	activeStorage *storage.ActiveStorage,
) module.Module {
	// Initialize service with active storage
	service := NewProfileService(db, logger, activeStorage)
	controller := NewProfileController(service, logger)

	usersModule := &UserModule{
		DB:            db,
		Controller:    controller,
		Service:       service,
		Logger:        logger,
		ActiveStorage: activeStorage,
	}

	return usersModule
}

func (m *UserModule) Routes(router *router.RouterGroup) {
	m.Controller.Routes(router)
}

func (m *UserModule) Migrate() error {
	err := m.DB.AutoMigrate(&User{})
	if err != nil {
		m.Logger.Error("Migration failed", logger.String("error", err.Error()))
		return err
	}

	// Seed default admin user
	if err := m.seedDefaultUser(); err != nil {
		m.Logger.Error("Failed to seed default user", logger.String("error", err.Error()))
		return err
	}

	return nil
}

// seedDefaultUser creates a default admin user if no users exist
func (m *UserModule) seedDefaultUser() error {
	var count int64
	if err := m.DB.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}

	// Only seed if no users exist
	if count > 0 {
		return nil
	}

	m.Logger.Info("No users found, creating default admin user...")

	// Hash password using bcrypt (same as authentication service)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create default admin user
	defaultUser := &User{
		Username:  "admin",
		Email:     "admin@base.al",
		FirstName: "Admin",
		LastName:  "User",
		Password:  string(hashedPassword),
		RoleId:    1, // Super Admin role ID
	}

	if err := m.DB.Create(defaultUser).Error; err != nil {
		return err
	}

	m.Logger.Info("Default admin user created successfully",
		logger.String("username", "admin"),
		logger.String("email", "admin@base.al"),
		logger.String("password", "admin123"))

	return nil
}

func (m *UserModule) GetModels() []any {
	return []any{
		&User{},
	}
}

func (m *UserModule) GetModelNames() []string {
	models := m.GetModels()
	names := make([]string, len(models))
	for i, model := range models {
		names[i] = m.DB.Model(model).Statement.Table
	}
	return names
}
