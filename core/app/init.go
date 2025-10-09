package app

import (
	"base/core/app/activities"
	"base/core/app/authentication"
	"base/core/app/authorization"
	"base/core/app/employees"
	"base/core/app/media"
	"base/core/app/notifications"
	"base/core/app/oauth"
	"base/core/app/profile"
	"base/core/app/search"
	"base/core/app/settings"
	"base/core/module"
	"base/core/scheduler"
	"base/core/translation"
)

// CoreModules implements module.CoreModuleProvider interface
type CoreModules struct {
	SearchRegistry *search.SearchRegistry
}

// GetCoreModules returns the list of core modules to initialize
// This is the only function that needs to be updated when adding new core modules
func (cm *CoreModules) GetCoreModules(deps module.Dependencies) map[string]module.Module {
	modules := make(map[string]module.Module)

	// Core modules - essential system functionality
	modules["users"] = profile.NewUserModule(
		deps.DB,
		deps.Router,
		deps.Logger,
		deps.Storage,
	)

	modules["media"] = media.NewMediaModule(
		deps.DB,
		deps.Router,
		deps.Storage,
		deps.Emitter,
		deps.Logger,
	)

	modules["authentication"] = authentication.NewAuthenticationModule(
		deps.DB,
		deps.Router, // Will be handled by orchestrator to use AuthRouter
		deps.EmailSender,
		deps.Logger,
		deps.Emitter,
	)

	modules["oauth"] = oauth.NewOAuthModule(
		deps.DB,
		deps.Router,
		deps.Logger,
		deps.Storage,
	)

	modules["authorization"] = authorization.NewAuthorizationModule(
		deps.DB,
		deps.Router, // Will be handled by orchestrator to use AuthRouter
		deps.Logger,
	)

	modules["translation"] = translation.NewTranslationModule(
		deps.DB,
		deps.Router,
		deps.Logger,
		deps.Emitter,
		deps.Storage,
	)

	modules["scheduler"] = scheduler.NewSchedulerModule(
		deps.DB,
		deps.Router,
		deps.Logger,
		deps.Emitter,
	)

	// Admin template essential modules
	modules["settings"] = settings.Init(deps)
	modules["employees"] = employees.Init(deps)

	// Initialize search with registry (can be nil, will create empty registry)
	modules["search"] = search.Init(deps, cm.SearchRegistry)

	modules["notifications"] = notifications.Init(deps)
	modules["activities"] = activities.Init(deps)

	return modules
}

// NewCoreModules creates a new core modules provider
func NewCoreModules(searchRegistry *search.SearchRegistry) *CoreModules {
	return &CoreModules{
		SearchRegistry: searchRegistry,
	}
}
