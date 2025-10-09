package activities

import (
	"net/http"
	"strconv"
	"strings"

	"base/core/router"
	"base/core/storage"
	"base/core/types"
)

type ActivityController struct {
	Service *ActivityService
	Storage *storage.ActiveStorage
}

func NewActivityController(service *ActivityService, storage *storage.ActiveStorage) *ActivityController {
	return &ActivityController{
		Service: service,
		Storage: storage,
	}
}

func (c *ActivityController) Routes(router *router.RouterGroup) {
	// Main CRUD endpoints - specific routes MUST come before parameterized routes
	router.GET("/activities", c.List)             // Paginated list
	router.POST("/activities", c.Create)          // Create
	router.GET("/activities/all", c.ListAll)      // Unpaginated list - MUST be before /:id
	router.GET("/activities/recent", c.GetRecent) // Get recent activities - MUST be before /:id
	router.GET("/activities/:id", c.Get)          // Get by ID - MUST be after /all
	router.PUT("/activities/:id", c.Update)       // Update
	router.DELETE("/activities/:id", c.Delete)    // Delete

	//Upload endpoints for each file field
}

// CreateActivity godoc
// @Summary Create a new Activity
// @Description Create a new Activity with the input payload
// @Tags Core/Activity
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param activities body CreateActivityRequest true "Create Activity request"
// @Success 201 {object} ActivityResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /activities [post]
func (c *ActivityController) Create(ctx *router.Context) error {
	var req CreateActivityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	item, err := c.Service.Create(&req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to create item: " + err.Error()})
	}

	return ctx.JSON(http.StatusCreated, item.ToResponse())
}

// GetActivity godoc
// @Summary Get a Activity
// @Description Get a Activity by its id
// @Tags Core/Activity
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Activity id"
// @Success 200 {object} ActivityResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /activities/{id} [get]
func (c *ActivityController) Get(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	item, err := c.Service.GetById(uint(id))
	if err != nil {
		return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "Item not found"})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// ListActivities godoc
// @Summary List activities
// @Description Get a list of activities
// @Tags Core/Activity
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Number of items per page"
// @Param sort query string false "Sort field (id, created_at, updated_at,user_id,entity_type,entity_id,action,description,metadata,ip_address,user_agent,)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /activities [get]
func (c *ActivityController) List(ctx *router.Context) error {
	var page, limit *int
	var sortBy, sortOrder *string

	// Parse page parameter
	if pageStr := ctx.Query("page"); pageStr != "" {
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = &pageNum
		} else {
			return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid page number"})
		}
	}

	// Parse limit parameter
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if limitNum, err := strconv.Atoi(limitStr); err == nil && limitNum > 0 {
			limit = &limitNum
		} else {
			return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid limit number"})
		}
	}

	// Parse sort parameters
	if sortStr := ctx.Query("sort"); sortStr != "" {
		sortBy = &sortStr
	}

	if orderStr := ctx.Query("order"); orderStr != "" {
		if orderStr == "asc" || orderStr == "desc" {
			sortOrder = &orderStr
		} else {
			return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid sort order. Use 'asc' or 'desc'"})
		}
	}

	paginatedResponse, err := c.Service.GetAll(page, limit, sortBy, sortOrder)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch items: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, paginatedResponse)
}

// ListAllActivities godoc
// @Summary List all activities for select options
// @Description Get a simplified list of all activities with id and name only (for dropdowns/select boxes)
// @Tags Core/Activity
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {array} ActivitySelectOption
// @Failure 500 {object} types.ErrorResponse
// @Router /activities/all [get]
func (c *ActivityController) ListAll(ctx *router.Context) error {
	items, err := c.Service.GetAllForSelect()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch select options: " + err.Error()})
	}

	// Convert to select options
	var selectOptions []*ActivitySelectOption
	for _, item := range items {
		selectOptions = append(selectOptions, item.ToSelectOption())
	}

	return ctx.JSON(http.StatusOK, selectOptions)
}

// UpdateActivity godoc
// @Summary Update a Activity
// @Description Update a Activity by its id
// @Tags Core/Activity
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Activity id"
// @Param activities body UpdateActivityRequest true "Update Activity request"
// @Success 200 {object} ActivityResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /activities/{id} [put]
func (c *ActivityController) Update(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	var req UpdateActivityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	item, err := c.Service.Update(uint(id), &req)
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "Item not found"})
		}
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to update item: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// DeleteActivity godoc
// @Summary Delete a Activity
// @Description Delete a Activity by its id
// @Tags Core/Activity
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Activity id"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /activities/{id} [delete]
func (c *ActivityController) Delete(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	if err := c.Service.Delete(uint(id)); err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "Item not found"})
		}
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to delete item: " + err.Error()})
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// GetRecent godoc
// @Summary Get recent activities
// @Description Get the most recent activities
// @Tags Core/Activity
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param limit query int false "Number of activities to return (default 10)"
// @Success 200 {array} ActivityResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /activities/recent [get]
func (c *ActivityController) GetRecent(ctx *router.Context) error {
	limitStr := ctx.Query("limit")
	limit := 10 // default

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	activities, err := c.Service.GetRecentActivities(limit)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get recent activities: " + err.Error()})
	}

	// Convert to response format
	responses := make([]*ActivityResponse, len(activities))
	for i, activity := range activities {
		responses[i] = activity.ToResponse()
	}

	return ctx.JSON(http.StatusOK, responses)
}
