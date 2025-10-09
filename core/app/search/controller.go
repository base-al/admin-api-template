package search

import (
	"net/http"
	"strconv"
	"time"

	"base/core/router"
	"base/core/storage"
	"base/core/types"
)

type SearchController struct {
	Service *SearchService
	Storage *storage.ActiveStorage
}

func NewSearchController(service *SearchService, storage *storage.ActiveStorage) *SearchController {
	return &SearchController{
		Service: service,
		Storage: storage,
	}
}

func (c *SearchController) Routes(router *router.RouterGroup) {
	// Global search endpoint
	router.GET("/search", c.Search)
}

// Search godoc
// @Summary Global search across modules
// @Description Search across multiple modules (customers, employees, business_customers, etc.)
// @Tags Global/Search
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param q query string true "Search query (minimum 2 characters)" example("john")
// @Param modules query string false "Comma-separated modules to search" example("customer,employee,business_customer")
// @Param limit query int false "Results per module (default: 10)" example(20)
// @Success 200 {object} search.SearchResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /search [get]
func (c *SearchController) Search(ctx *router.Context) error {
	startTime := time.Now()

	// Manually parse query parameters since bindData is a placeholder
	query := ctx.Query("q")
	if query == "" {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Search query (q) is required"})
	}
	if len(query) < 2 {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Search query must be at least 2 characters"})
	}

	modules := ctx.Query("modules")
	limitStr := ctx.Query("limit")
	limit := 10
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Perform search
	response, err := c.Service.GlobalSearch(query, modules, limit)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Search failed: " + err.Error()})
	}

	// Add duration
	response.Duration = time.Since(startTime).String()

	return ctx.JSON(http.StatusOK, response)
}
