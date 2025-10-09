package search

import (
	"fmt"
	"strings"

	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"

	"gorm.io/gorm"
)

type SearchService struct {
	DB       *gorm.DB
	Emitter  *emitter.Emitter
	Storage  *storage.ActiveStorage
	Logger   logger.Logger
	Registry *SearchRegistry
}

func NewSearchService(db *gorm.DB, emitter *emitter.Emitter, storage *storage.ActiveStorage, logger logger.Logger, registry *SearchRegistry) *SearchService {
	return &SearchService{
		DB:       db,
		Logger:   logger,
		Emitter:  emitter,
		Storage:  storage,
		Registry: registry,
	}
}

// GlobalSearch performs search across multiple modules using the registry
func (s *SearchService) GlobalSearch(query, modules string, limit int) (*SearchResponse, error) {
	response := &SearchResponse{
		Query:   query,
		Results: make(map[string][]SearchResult),
		Modules: []string{},
		Total:   0,
	}

	// Default limit
	if limit <= 0 {
		limit = 10
	}

	// Parse modules to search
	var modulesToSearch []string
	if modules == "" {
		// Default to all registered modules
		modulesToSearch = s.Registry.GetNames()
	} else {
		modulesToSearch = strings.Split(modules, ",")
		// Trim whitespace
		for i, module := range modulesToSearch {
			modulesToSearch[i] = strings.TrimSpace(module)
		}
	}

	// Search each module
	for _, moduleName := range modulesToSearch {
		config, exists := s.Registry.Get(moduleName)
		if !exists {
			s.Logger.Warn("Search module not registered",
				logger.String("module", moduleName))
			continue
		}

		results, err := s.searchWithConfig(config, query, limit)
		if err != nil {
			s.Logger.Error("Failed to search module",
				logger.String("module", moduleName),
				logger.String("error", err.Error()))
			continue
		}

		if len(results) > 0 {
			response.Results[moduleName] = results
			response.Modules = append(response.Modules, moduleName)
			response.Total += len(results)
		}
	}

	return response, nil
}

// searchWithConfig searches using a registered search config
func (s *SearchService) searchWithConfig(config *SearchConfig, query string, limit int) ([]SearchResult, error) {
	// If custom search function is provided, use it
	if config.CustomSearchFunc != nil {
		return config.CustomSearchFunc(s.DB, query, limit)
	}

	// Default search: build dynamic LIKE query for all fields
	return s.defaultSearch(config, query, limit)
}

// defaultSearch performs a default LIKE search across configured fields
func (s *SearchService) defaultSearch(config *SearchConfig, query string, limit int) ([]SearchResult, error) {
	if len(config.Fields) == 0 {
		s.Logger.Warn("No search fields configured for module",
			logger.String("module", config.Name))
		return []SearchResult{}, nil
	}

	// Build WHERE clause with OR conditions for each field
	var whereClauses []string
	var whereArgs []interface{}

	for _, field := range config.Fields {
		whereClauses = append(whereClauses, field+" LIKE ?")
		whereArgs = append(whereArgs, "%"+query+"%")
	}

	whereClause := strings.Join(whereClauses, " OR ")

	// Execute query - get all columns as we'll use the model's ToSearchResult method
	rows, err := s.DB.Table(config.Table).
		Where("deleted_at IS NULL").
		Where(whereClause, whereArgs...).
		Limit(limit).
		Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		// Create a map to hold the row data
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			continue
		}

		// Build a map of column -> value
		rowData := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowData[col] = string(b)
			} else {
				rowData[col] = val
			}
		}

		// Create a basic search result
		// Models can implement ToSearchResult for custom formatting
		result := s.createBasicSearchResult(config, rowData)
		results = append(results, result)
	}

	return results, nil
}

// createBasicSearchResult creates a basic search result from row data
func (s *SearchService) createBasicSearchResult(config *SearchConfig, rowData map[string]interface{}) SearchResult {
	// Get ID
	var id uint
	if idVal, ok := rowData["id"]; ok {
		switch v := idVal.(type) {
		case int64:
			id = uint(v)
		case uint:
			id = v
		case uint64:
			id = uint(v)
		}
	}

	// Build title from first available search field
	var title string
	if len(config.Fields) > 0 {
		if val, ok := rowData[config.Fields[0]]; ok {
			title = s.toString(val)
		}
	}

	// Build subtitle from second field if available
	var subtitle string
	if len(config.Fields) > 1 {
		if val, ok := rowData[config.Fields[1]]; ok {
			subtitle = s.toString(val)
		}
	}

	// Build description from remaining fields
	var description string
	if len(config.Fields) > 2 {
		if val, ok := rowData[config.Fields[2]]; ok {
			description = s.toString(val)
		}
	}

	return SearchResult{
		Id:          id,
		Type:        config.Type,
		Title:       title,
		Subtitle:    subtitle,
		Description: description,
		URL:         "/app/" + config.Name + "/" + s.toString(id),
		Metadata:    rowData,
	}
}

// toString converts interface{} to string
func (s *SearchService) toString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int64, uint, uint64:
		return strings.TrimSpace(strings.ReplaceAll(fmt.Sprintf("%v", v), "\n", " "))
	default:
		return strings.TrimSpace(strings.ReplaceAll(fmt.Sprintf("%v", v), "\n", " "))
	}
}
