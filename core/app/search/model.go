package search

// Search represents a search entity
type SearchResponse struct {
	Query    string                    `json:"query"`    // Original search query
	Total    int                       `json:"total"`    // Total results across all modules
	Results  map[string][]SearchResult `json:"results"`  // Results grouped by module
	Modules  []string                  `json:"modules"`  // Modules that were searched
	Duration string                    `json:"duration"` // Search duration
}

type SearchResult struct {
	Id          uint   `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Metadata    any    `json:"metadata"`
}

type SearchRequest struct {
	Query   string `form:"q" binding:"required,min=2" example:"john"`                       // Search query (minimum 2 characters)
	Modules string `form:"modules,omitempty" example:"customer,employee,business_customer"` // Comma-separated modules to search
	Limit   int    `form:"limit,omitempty" example:"20"`                                    // Results per module (default: 10)
}
