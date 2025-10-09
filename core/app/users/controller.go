package users

import (
	"base/core/app/authorization"
	"base/core/logger"
	"base/core/router"
	"base/core/storage"
	"base/core/types"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserController struct {
	service *UserService
	storage *storage.ActiveStorage
	logger  logger.Logger
}

func NewUserController(service *UserService, storage *storage.ActiveStorage, logger logger.Logger) *UserController {
	return &UserController{
		service: service,
		storage: storage,
		logger:  logger,
	}
}

func (c *UserController) Routes(router *router.RouterGroup) {
	// Profile endpoints - for authenticated user (no role restriction)
	router.GET("/profile", c.GetProfile)
	router.PUT("/profile", c.UpdateProfile)
	router.PUT("/profile/avatar", c.UpdateAvatar)
	router.PUT("/profile/password", c.UpdatePassword)

	// User management endpoints - admin only
	adminOnlyMiddleware := authorization.RequireRole("Admin")
	usersGroup := router.Group("/users")
	usersGroup.Use(adminOnlyMiddleware)

	usersGroup.GET("", c.List)                  // Paginated list
	usersGroup.POST("", c.Create)               // Create
	usersGroup.GET("/all", c.ListAll)           // Unpaginated list
	usersGroup.GET("/:id", c.Get)               // Get by ID
	usersGroup.PUT("/:id", c.Update)            // Update
	usersGroup.PUT("/:id/password", c.ChangePassword) // Change password
	usersGroup.GET("/:id/tasks", c.GetUserTasks)      // Get tasks
	usersGroup.DELETE("/:id", c.Delete)               // Delete
}

// Profile Endpoints (no admin restriction)

// GetProfile godoc
// @Summary Get profile from Authenticated User Token
// @Description Get profile by Bearer Token
// @Security ApiKeyAuth
// @Security BearerAuth
// @Tags Core/Profile
// @Accept json
// @Produce json
// @Success 200 {object} UserResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /profile [get]
func (c *UserController) GetProfile(ctx *router.Context) error {
	id := ctx.GetUint("user_id")
	c.logger.Debug("Getting user", logger.Uint("user_id", id))
	if id == 0 {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid user Id"})
	}

	item, err := c.service.GetById(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		}
		c.logger.Error("Failed to get user", logger.Uint("user_id", id))
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch user"})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// UpdateProfile godoc
// @Summary Update profile from Authenticated User Token
// @Description Update profile by Bearer Token
// @Security ApiKeyAuth
// @Security BearerAuth
// @Tags Core/Profile
// @Accept json
// @Produce json
// @Param input body UpdateUserRequest true "Update Request"
// @Success 200 {object} UserResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /profile [put]
func (c *UserController) UpdateProfile(ctx *router.Context) error {
	id := ctx.GetUint("user_id")
	if id == 0 {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid Id format"})
	}

	var req UpdateUserRequest
	if err := ctx.ShouldBind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid input: " + err.Error()})
	}

	item, err := c.service.Update(id, &req)
	if err != nil {
		c.logger.Error("Failed to update user", logger.Uint("user_id", id))
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to update user: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// UpdateAvatar godoc
// @Summary Update profile avatar from Authenticated User Token
// @Description Update profile avatar by Bearer Token
// @Security ApiKeyAuth
// @Security BearerAuth
// @Tags Core/Profile
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Avatar file"
// @Success 200 {object} UserResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /profile/avatar [put]
func (c *UserController) UpdateAvatar(ctx *router.Context) error {
	id := ctx.GetUint("user_id")
	if id == 0 {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid Id format"})
	}

	file, err := ctx.FormFile("avatar")
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Failed to get avatar file: " + err.Error()})
	}

	updatedUser, err := c.service.UpdateAvatar(ctx, id, file)
	if err != nil {
		c.logger.Error("Failed to update avatar", logger.Uint("user_id", id))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		}
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to update avatar: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, updatedUser.ToResponse())
}

// UpdatePassword godoc
// @Summary Update profile password from Authenticated User Token
// @Description Update profile password by Bearer Token
// @Security ApiKeyAuth
// @Security BearerAuth
// @Tags Core/Profile
// @Accept json
// @Produce json
// @Param input body UpdatePasswordRequest true "Update Password Request"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /profile/password [put]
func (c *UserController) UpdatePassword(ctx *router.Context) error {
	id := ctx.GetUint("user_id")
	if id == 0 {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid user Id"})
	}

	var req UpdatePasswordRequest
	if err := ctx.ShouldBind(&req); err != nil {
		c.logger.Error("Failed to bind password update request")
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid input: " + err.Error()})
	}

	if len(req.NewPassword) < 6 {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "New password must be at least 6 characters long"})
	}

	err := c.service.UpdatePassword(id, &req)
	if err != nil {
		c.logger.Error("Failed to update password", logger.Uint("user_id", id))
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return ctx.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "Current password is incorrect"})
		default:
			return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to update password"})
		}
	}

	return ctx.JSON(http.StatusOK, types.SuccessResponse{Message: "Password updated successfully"})
}

// User Management Endpoints (Admin only)

// Create godoc
// @Summary Create a new User
// @Description Create a new User with the input payload (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param users body CreateUserRequest true "Create User request"
// @Success 201 {object} UserResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /users [post]
func (c *UserController) Create(ctx *router.Context) error {
	var req CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	item, err := c.service.Create(&req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to create user: " + err.Error()})
	}

	return ctx.JSON(http.StatusCreated, item.ToResponse())
}

// Get godoc
// @Summary Get a User
// @Description Get a User by its id (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User id"
// @Success 200 {object} UserResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /users/{id} [get]
func (c *UserController) Get(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	item, err := c.service.GetById(uint(id))
	if err != nil {
		return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// List godoc
// @Summary List users
// @Description Get a paginated list of users (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Number of items per page"
// @Param sort query string false "Sort field (id, created_at, updated_at, first_name, last_name, username, phone, email, role_id)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /users [get]
func (c *UserController) List(ctx *router.Context) error {
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

	paginatedResponse, err := c.service.GetAll(page, limit, sortBy, sortOrder)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch users: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, paginatedResponse)
}

// ListAll godoc
// @Summary List all users for select options
// @Description Get a simplified list of all users with id and name only (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {array} UserSelectOption
// @Failure 500 {object} types.ErrorResponse
// @Router /users/all [get]
func (c *UserController) ListAll(ctx *router.Context) error {
	items, err := c.service.GetAllForSelect()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch select options: " + err.Error()})
	}

	// Convert to select options
	var selectOptions []*UserSelectOption
	for _, item := range items {
		selectOptions = append(selectOptions, item.ToSelectOption())
	}

	return ctx.JSON(http.StatusOK, selectOptions)
}

// Update godoc
// @Summary Update a User
// @Description Update a User by its id (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User id"
// @Param users body UpdateUserRequest true "Update User request"
// @Success 200 {object} UserResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /users/{id} [put]
func (c *UserController) Update(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	var req UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	item, err := c.service.Update(uint(id), &req)
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		}
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to update user: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// Delete godoc
// @Summary Delete a User
// @Description Delete a User by its id (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User id"
// @Success 204 {object} nil
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /users/{id} [delete]
func (c *UserController) Delete(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	if err := c.service.Delete(uint(id)); err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		}
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to delete user: " + err.Error()})
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// ChangePassword godoc
// @Summary Change user password
// @Description Change the password for a specific user (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User id"
// @Param request body ChangePasswordRequest true "Change password request"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /users/{id}/password [put]
func (c *UserController) ChangePassword(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	var req ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	if err := c.service.ChangePassword(uint(id), req.NewPassword, req.CurrentPassword); err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return ctx.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
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

// GetUserTasks godoc
// @Summary Get tasks assigned to a user
// @Description Get all tasks assigned to a specific user (Admin only)
// @Tags Core/Users
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "User id"
// @Success 200 {array} interface{}
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /users/{id}/tasks [get]
func (c *UserController) GetUserTasks(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	tasks, err := c.service.GetUserTasks(uint(id))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch user tasks: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"data": tasks})
}
