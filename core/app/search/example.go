package search

// Example implementation of SearchableModel interface
//
// To make any model searchable in the global search, implement the SearchableModel interface:
//
// type Product struct {
//     ID          uint   `json:"id"`
//     Name        string `json:"name"`
//     Description string `json:"description"`
//     SKU         string `json:"sku"`
//     Price       float64 `json:"price"`
// }
//
// func (p *Product) GetSearchFields() []string {
//     return []string{"name", "description", "sku"}
// }
//
// func (p *Product) GetSearchTable() string {
//     return "products"
// }
//
// func (p *Product) GetSearchType() string {
//     return "product"
// }
//
// func (p *Product) ToSearchResult() SearchResult {
//     return SearchResult{
//         Id:          p.ID,
//         Type:        "product",
//         Title:       p.Name,
//         Subtitle:    p.SKU,
//         Description: p.Description,
//         URL:         fmt.Sprintf("/app/products/%d", p.ID),
//         Metadata:    p,
//     }
// }
//
// Then register it in app/init.go GetSearchRegistry():
//
// func GetSearchRegistry() *search.SearchRegistry {
//     registry := search.NewSearchRegistry()
//
//     // Register your searchable models
//     registry.Register("products", &models.Product{})
//     registry.Register("orders", &models.Order{})
//
//     return registry
// }
//
// The search API will automatically search across all registered models:
// GET /api/search?q=john
// Returns:
// {
//     "query": "john",
//     "total": 15,
//     "results": {
//         "products": [...],
//         "orders": [...],
//         "employees": [...]
//     },
//     "modules": ["products", "orders", "employees"]
// }
