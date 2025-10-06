package search

import (
	"base/core/module"
	"base/core/router"

	"gorm.io/gorm"
)

type Module struct {
	module.DefaultModule
	DB         *gorm.DB
	Service    *SearchService
	Controller *SearchController
}

// Init creates and initializes the Search module with all dependencies
func Init(deps module.Dependencies) module.Module {
	// Initialize service and controller
	service := NewSearchService(deps.DB, deps.Emitter, deps.Storage, deps.Logger)
	controller := NewSearchController(service, deps.Storage)

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
