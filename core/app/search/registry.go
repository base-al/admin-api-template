package search

import "gorm.io/gorm"

// SearchableModel interface that models must implement to be searchable
type SearchableModel interface {
	// GetSearchFields returns the fields to search in (e.g., "name", "email", "description")
	GetSearchFields() []string

	// GetSearchTable returns the database table name
	GetSearchTable() string

	// GetSearchType returns the type identifier for the search result (e.g., "product", "employee")
	GetSearchType() string

	// ToSearchResult converts the model instance to a SearchResult
	ToSearchResult() SearchResult
}

// SearchConfig represents the configuration for searching a specific model
type SearchConfig struct {
	// Model is an instance of the searchable model (used for type information)
	Model SearchableModel

	// Name is the module name (e.g., "products", "orders")
	Name string

	// Fields are the database columns to search in
	Fields []string

	// Table is the database table name
	Table string

	// Type is the search result type identifier
	Type string

	// CustomSearchFunc allows custom search logic (optional)
	// If provided, this function will be used instead of the default LIKE search
	CustomSearchFunc func(db *gorm.DB, query string, limit int) ([]SearchResult, error)
}

// SearchRegistry holds all registered searchable models
type SearchRegistry struct {
	configs map[string]*SearchConfig
}

// NewSearchRegistry creates a new search registry
func NewSearchRegistry() *SearchRegistry {
	return &SearchRegistry{
		configs: make(map[string]*SearchConfig),
	}
}

// SimpleSearchConfig is a simplified configuration for quick registration
type SimpleSearchConfig struct {
	Table  string   // Database table name
	Fields []string // Fields to search in
	Type   string   // Type identifier for results (optional, defaults to table name)
}

// RegisterSimple adds a model with minimal configuration
// Example: registry.RegisterSimple("products", search.SimpleSearchConfig{
//     Table:  "products",
//     Fields: []string{"name", "description", "sku"},
// })
func (r *SearchRegistry) RegisterSimple(name string, cfg SimpleSearchConfig) {
	// Default type to name if not provided
	if cfg.Type == "" {
		cfg.Type = name
	}

	config := &SearchConfig{
		Model:  nil, // No model instance needed for simple registration
		Name:   name,
		Fields: cfg.Fields,
		Table:  cfg.Table,
		Type:   cfg.Type,
	}
	r.configs[name] = config
}

// Register adds a searchable model to the registry
func (r *SearchRegistry) Register(name string, model SearchableModel) {
	config := &SearchConfig{
		Model:  model,
		Name:   name,
		Fields: model.GetSearchFields(),
		Table:  model.GetSearchTable(),
		Type:   model.GetSearchType(),
	}
	r.configs[name] = config
}

// RegisterWithCustomSearch adds a searchable model with custom search function
func (r *SearchRegistry) RegisterWithCustomSearch(name string, model SearchableModel, searchFunc func(db *gorm.DB, query string, limit int) ([]SearchResult, error)) {
	config := &SearchConfig{
		Model:            model,
		Name:             name,
		Fields:           model.GetSearchFields(),
		Table:            model.GetSearchTable(),
		Type:             model.GetSearchType(),
		CustomSearchFunc: searchFunc,
	}
	r.configs[name] = config
}

// Get retrieves a search config by name
func (r *SearchRegistry) Get(name string) (*SearchConfig, bool) {
	config, exists := r.configs[name]
	return config, exists
}

// GetAll returns all registered search configs
func (r *SearchRegistry) GetAll() map[string]*SearchConfig {
	return r.configs
}

// GetNames returns all registered module names
func (r *SearchRegistry) GetNames() []string {
	names := make([]string, 0, len(r.configs))
	for name := range r.configs {
		names = append(names, name)
	}
	return names
}
