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
	Registry   *SearchRegistry
}

// Init creates and initializes the Search module with all dependencies
// Pass a registry from app/init.go to configure searchable models
func Init(deps module.Dependencies, registry *SearchRegistry) module.Module {
	// If no registry provided, create an empty one
	if registry == nil {
		registry = NewSearchRegistry()
	}

	// Initialize service and controller
	service := NewSearchService(deps.DB, deps.Emitter, deps.Storage, deps.Logger, registry)
	controller := NewSearchController(service, deps.Storage)

	// Create module
	mod := &Module{
		DB:         deps.DB,
		Service:    service,
		Controller: controller,
		Registry:   registry,
	}

	return mod
}

// Routes registers the module routes
func (m *Module) Routes(router *router.RouterGroup) {
	m.Controller.Routes(router)
}
