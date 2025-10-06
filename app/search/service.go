package search

import (
	"strings"

	"base/app/employees"
	"base/app/models"
	"base/core/emitter"
	"base/core/logger"
	"base/core/storage"

	"gorm.io/gorm"
)

type SearchService struct {
	DB               *gorm.DB
	Emitter          *emitter.Emitter
	Storage          *storage.ActiveStorage
	Logger           logger.Logger
	EmployeesService *employees.EmployeeService
}

func NewSearchService(db *gorm.DB, emitter *emitter.Emitter, storage *storage.ActiveStorage, logger logger.Logger) *SearchService {
	return &SearchService{
		DB:      db,
		Logger:  logger,
		Emitter: emitter,
		Storage: storage,
		// Initialize module services
		EmployeesService: employees.NewEmployeeService(db, emitter, storage, logger),
	}
}

// GlobalSearch performs search across multiple modules
func (s *SearchService) GlobalSearch(query, modules string, limit int) (*models.SearchResponse, error) {
	response := &models.SearchResponse{
		Query:   query,
		Results: make(map[string][]models.SearchResult),
		Modules: []string{},
		Total:   0,
	}

	// Parse modules to search
	var modulesToSearch []string
	if modules == "" {
		// Default to all modules
		modulesToSearch = []string{"employee"}
	} else {
		modulesToSearch = strings.Split(modules, ",")
		// Trim whitespace
		for i, module := range modulesToSearch {
			modulesToSearch[i] = strings.TrimSpace(module)
		}
	}

	// Search each module
	for _, module := range modulesToSearch {
		results, err := s.searchModule(module, query, limit)
		if err != nil {
			s.Logger.Error("Failed to search module",
				logger.String("module", module),
				logger.String("error", err.Error()))
			continue
		}

		if len(results) > 0 {
			response.Results[module] = results
			response.Modules = append(response.Modules, module)
			response.Total += len(results)
		}
	}

	return response, nil
}

// searchModule searches within a specific module
func (s *SearchService) searchModule(module, query string, limit int) ([]models.SearchResult, error) {
	switch module {
	case "employee":
		return s.searchEmployees(query, limit)
	default:
		s.Logger.Warn("Unknown search module", logger.String("module", module))
		return []models.SearchResult{}, nil
	}
}

// searchEmployees searches in employees
func (s *SearchService) searchEmployees(query string, limit int) ([]models.SearchResult, error) {
	var employees []struct {
		ID        uint   `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		Phone     string `json:"phone"`
	}

	dbQuery := s.DB.Table("users").
		Select("id, first_name, last_name, email, username, phone").
		Where("deleted_at IS NULL").
		Where("CONCAT(first_name, ' ', last_name) LIKE ? OR email LIKE ? OR username LIKE ?",
			"%"+query+"%", "%"+query+"%", "%"+query+"%").
		Limit(limit)

	if err := dbQuery.Find(&employees).Error; err != nil {
		return nil, err
	}

	results := make([]models.SearchResult, len(employees))
	for i, employee := range employees {
		name := strings.TrimSpace(employee.FirstName + " " + employee.LastName)
		if name == "" {
			name = employee.Username
		}

		results[i] = models.SearchResult{
			Id:          employee.ID,
			Type:        "employee",
			Title:       name,
			Subtitle:    employee.Email,
			Description: "Username: " + employee.Username,
			URL:         "/app/employees/" + string(rune(employee.ID)),
			Metadata:    employee,
		}
	}

	return results, nil
}
