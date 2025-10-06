package posts

import (
	"net/http"
	"strconv"
	"strings"

	"base/app/models"
	"base/core/router"
	"base/core/storage"
	"base/core/types"
)

type PostController struct {
	Service *PostService
	Storage *storage.ActiveStorage
}

func NewPostController(service *PostService, storage *storage.ActiveStorage) *PostController {
	return &PostController{
		Service: service,
		Storage: storage,
	}
}

func (c *PostController) Routes(router *router.RouterGroup) {
	// Main CRUD endpoints - specific routes MUST come before parameterized routes
	router.GET("/posts", c.List)          // Paginated list
	router.POST("/posts", c.Create)       // Create
	router.GET("/posts/all", c.ListAll)   // Unpaginated list - MUST be before /:id
	router.GET("/posts/:id", c.Get)       // Get by ID - MUST be after /all
	router.PUT("/posts/:id", c.Update)    // Update
	router.DELETE("/posts/:id", c.Delete) // Delete

	//Upload endpoints for each file field
	router.POST("/posts/:id/featured-image", c.UploadFeaturedImage)
	router.DELETE("/posts/:id/featured-image", c.RemoveFeaturedImage)
}

// CreatePost godoc
// @Summary Create a new Post
// @Description Create a new Post with the input payload
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param posts body models.CreatePostRequest true "Create Post request"
// @Success 201 {object} models.PostResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /posts [post]
func (c *PostController) Create(ctx *router.Context) error {
	var req models.CreatePostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
	}

	item, err := c.Service.Create(&req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to create item: " + err.Error()})
	}

	return ctx.JSON(http.StatusCreated, item.ToResponse())
}

// GetPost godoc
// @Summary Get a Post
// @Description Get a Post by its id
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Post id"
// @Success 200 {object} models.PostResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Router /posts/{id} [get]
func (c *PostController) Get(ctx *router.Context) error {
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

// ListPosts godoc
// @Summary List posts
// @Description Get a list of posts
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Number of items per page"
// @Param sort query string false "Sort field (id, created_at, updated_at,title,slug,content,excerpt,author_id,status,category,featured_image,published,featured,is_pinned,view_count,like_count,rating,published_at,scheduled_at,tags,metadata,)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /posts [get]
func (c *PostController) List(ctx *router.Context) error {
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

// ListAllPosts godoc
// @Summary List all posts for select options
// @Description Get a simplified list of all posts with id and name only (for dropdowns/select boxes)
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {array} models.PostSelectOption
// @Failure 500 {object} types.ErrorResponse
// @Router /posts/all [get]
func (c *PostController) ListAll(ctx *router.Context) error {
	items, err := c.Service.GetAllForSelect()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to fetch select options: " + err.Error()})
	}

	// Convert to select options
	var selectOptions []*models.PostSelectOption
	for _, item := range items {
		selectOptions = append(selectOptions, item.ToSelectOption())
	}

	return ctx.JSON(http.StatusOK, selectOptions)
}

// UpdatePost godoc
// @Summary Update a Post
// @Description Update a Post by its id
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Post id"
// @Param posts body models.UpdatePostRequest true "Update Post request"
// @Success 200 {object} models.PostResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /posts/{id} [put]
func (c *PostController) Update(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	var req models.UpdatePostRequest
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

// DeletePost godoc
// @Summary Delete a Post
// @Description Delete a Post by its id
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Post id"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /posts/{id} [delete]
func (c *PostController) Delete(ctx *router.Context) error {
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

// UploadFeaturedImage godoc
// @Summary Upload featured-image for Post
// @Description Upload a file for the Post's featured-image field
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Post id"
// @Param file formData file true "FeaturedImage file"
// @Success 200 {object} models.PostResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /posts/{id}/featured_image [post]
func (c *PostController) UploadFeaturedImage(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "No file uploaded"})
	}

	item, err := c.Service.UploadFeaturedImage(uint(id), file)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to upload featured-image: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// RemoveFeaturedImage godoc
// @Summary Remove featured-image from Post
// @Description Remove the file from the Post's featured-image field
// @Tags App/Post
// @Security ApiKeyAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Post id"
// @Success 200 {object} models.PostResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /posts/{id}/featured_image [delete]
func (c *PostController) RemoveFeaturedImage(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid id format"})
	}

	item, err := c.Service.RemoveFeaturedImage(uint(id))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to remove featured-image: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}
