package employees

import (
	"net/http"
	"strconv"
	"strings"

	"base/core/app/authorization"
	"base/core/router"
	"base/core/storage"
	"base/core/types"
)

type EmployeeController struct {
	Service *EmployeeService
	Storage *storage.ActiveStorage
}

func NewEmployeeController(service *EmployeeService, storage *storage.ActiveStorage) *EmployeeController {
	return &EmployeeController{
		Service: service,
		Storage: storage,
	}
}

func (c *EmployeeController) Routes(router *router.RouterGroup) {
	// Apply authorization middleware - only Admin can access employee endpoints
	adminOnlyMiddleware := authorization.RequireRole("Admin")

	// Create a sub-group for employee routes with admin-only middleware
	employeeGroup := router.Group("/employees")
	employeeGroup.Use(adminOnlyMiddleware)

	// Main CRUD endpoints - specific routes MUST come before parameterized routes
	employeeGroup.GET("", c.List)                        // Paginated list
	employeeGroup.POST("", c.Create)                     // Create
	employeeGroup.GET("/all", c.ListAll)                 // Unpaginated list - MUST be before /:id
	employeeGroup.GET("/:id", c.Get)                     // Get by ID - MUST be after /all
	employeeGroup.PUT("/:id", c.Update)                  // Update
	employeeGroup.PUT("/:id/password", c.ChangePassword) // Change password - MUST be before /:id
	employeeGroup.GET("/:id/tasks", c.GetEmployeeTasks) // Get tasks assigned to employee
	employeeGroup.DELETE("/:id", c.Delete)               // Delete

	//Upload endpoints for each file field
}

// CreateEmployee godoc
// @Summary Create a new Employee
// @Description Create a new Employee with the input payload
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param employees body CreateEmployeeRequest true "Create Employee request"
// @Success 201 {object} EmployeeResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /employees [post]
func (c *EmployeeController) Create(ctx *router.Context) error {
	var req CreateEmployeeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	item, err := c.Service.Create(&req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to create item: " + err.Error()})
	}

	return ctx.JSON(http.StatusCreated, item.ToResponse())
}

// GetEmployee godoc
// @Summary Get a Employee
// @Description Get a Employee by its id
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Employee id"
// @Success 200 {object} EmployeeResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /employees/{id} [get]
func (c *EmployeeController) Get(ctx *router.Context) error {
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

// ListEmployees godoc
// @Summary List employees
// @Description Get a list of employees
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Number of items per page"
// @Param sort query string false "Sort field (id, created_at, updated_at,first_name,last_name,username,phone,email,role_id,password,)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /employees [get]
func (c *EmployeeController) List(ctx *router.Context) error {
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

// ListAllEmployees godoc
// @Summary List all employees for select options
// @Description Get a simplified list of all employees with id and name only (for dropdowns/select boxes)
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {array} EmployeeSelectOption
// @Failure 500 {object} types.ErrorResponse
// @Router /employees/all [get]
func (c *EmployeeController) ListAll(ctx *router.Context) error {
	items, err := c.Service.GetAllForSelect()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch select options: " + err.Error()})
	}

	// Convert to select options
	var selectOptions []*EmployeeSelectOption
	for _, item := range items {
		selectOptions = append(selectOptions, item.ToSelectOption())
	}

	return ctx.JSON(http.StatusOK, selectOptions)
}

// UpdateEmployee godoc
// @Summary Update a Employee
// @Description Update a Employee by its id
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Employee id"
// @Param employees body UpdateEmployeeRequest true "Update Employee request"
// @Success 200 {object} EmployeeResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /employees/{id} [put]
func (c *EmployeeController) Update(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	var req UpdateEmployeeRequest
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

// DeleteEmployee godoc
// @Summary Delete a Employee
// @Description Delete a Employee by its id
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Employee id"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /employees/{id} [delete]
func (c *EmployeeController) Delete(ctx *router.Context) error {
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

// ChangePasswordRequest represents the request for changing employee password
type ChangePasswordRequest struct {
	NewPassword     string `json:"NewPassword" binding:"required"`
	CurrentPassword string `json:"CurrentPassword" binding:"required"`
}

// ChangePassword godoc
// @Summary Change employee password
// @Description Change the password for a specific employee
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Employee id"
// @Param request body ChangePasswordRequest true "Change password request"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /employees/{id}/password [put]
func (c *EmployeeController) ChangePassword(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	var req ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	if err := c.Service.ChangeEmployeePassword(uint(id), req.NewPassword, req.CurrentPassword); err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "Employee not found"})
		}
		if strings.Contains(err.Error(), "invalid current password") {
			return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid current password"})
		}
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to change password: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, types.SuccessResponse{
		Success: true,
		Message: "Password changed successfully",
	})
}

// GetEmployeeTasks godoc
// @Summary Get tasks assigned to an employee
// @Description Get all tasks assigned to a specific employee
// @Tags App/Employee
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Employee id"
// @Success 200 {array} interface{}
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /employees/{id}/tasks [get]
func (c *EmployeeController) GetEmployeeTasks(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	// Get tasks assigned to this employee
	tasks, err := c.Service.GetEmployeeTasks(uint(id))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch employee tasks: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"data": tasks})
}
