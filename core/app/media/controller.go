package media

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"base/core/logger"
	"base/core/router"
	"base/core/storage"
)

type MediaController struct {
	Service *MediaService
	Storage *storage.ActiveStorage
	Logger  logger.Logger
}

func NewMediaController(service *MediaService, storage *storage.ActiveStorage, logger logger.Logger) *MediaController {
	return &MediaController{
		Service: service,
		Storage: storage,
		Logger:  logger,
	}
}

func (c *MediaController) Routes(router *router.RouterGroup) {
	// Main CRUD endpoints
	router.GET("/media", c.List) // Paginated list
	router.POST("/media", c.Create)

	// Specific endpoints (must come before :id routes)
	router.GET("/media/all", c.ListAll) // Unpaginated list
	router.POST("/media/sync", c.SyncFromR2) // Sync from R2 bucket

	// Parameterized routes (must come last)
	router.GET("/media/:id", c.Get)
	router.PUT("/media/:id", c.Update)
	router.DELETE("/media/:id", c.Delete)

	// File management endpoints
	router.PUT("/media/:id/file", c.UpdateFile)
	router.DELETE("/media/:id/file", c.RemoveFile)
}

// Create godoc
// @Summary Create a new media item
// @Description Create a new media item with optional file upload
// @Tags Core/Media
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Media name"
// @Param type formData string true "Media type"
// @Param description formData string false "Media description"
// @Param file formData file false "Media file"
// @Success 201 {object} MediaResponse
// @Router /media [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) Create(ctx *router.Context) error {
	var req CreateMediaRequest

	contentType := ctx.Request.Header.Get("Content-Type")

	// Try to parse as JSON first, fall back to form data
	if strings.Contains(contentType, "application/json") {
		// Parse JSON request
		if err := ctx.ShouldBindJSON(&req); err != nil {
			return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
	} else {
		// Parse multipart form first
		if parseErr := ctx.Request.ParseMultipartForm(32 << 20); parseErr != nil {
			return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: parseErr.Error()})
		}

		// Manually parse all fields from multipart form
		req.Name = ctx.Request.FormValue("name")
		req.Type = ctx.Request.FormValue("type")
		req.Description = ctx.Request.FormValue("description")
		req.Folder = ctx.Request.FormValue("folder")
		req.Tags = ctx.Request.FormValue("tags")
		req.Metadata = ctx.Request.FormValue("metadata")

		// Parse parent_id if present
		if parentIdStr := ctx.Request.FormValue("parent_id"); parentIdStr != "" {
			if parentId, err := strconv.ParseUint(parentIdStr, 10, 32); err == nil {
				pid := uint(parentId)
				req.ParentId = &pid
			}
		}

		// Parse author_id if present
		if authorIdStr := ctx.Request.FormValue("author_id"); authorIdStr != "" {
			if authorId, err := strconv.ParseUint(authorIdStr, 10, 32); err == nil {
				aid := uint(authorId)
				req.AuthorId = &aid
			}
		}

		// Handle file upload if present
		if file, err := ctx.FormFile("file"); err == nil {
			req.File = file
		}
	}


	item, err := c.Service.Create(&req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusCreated, item.ToResponse())
}

// UpdateFile godoc
// @Summary Update media file
// @Description Update the file attached to a media item
// @Tags Core/Media
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Media Id"
// @Param file formData file true "Media file"
// @Success 200 {object} MediaResponse
// @Router /media/{id}/file [put]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) UpdateFile(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id parameter"})
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "file is required"})
	}

	item, err := c.Service.UpdateFile(ctx, uint(id), file)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// RemoveFile godoc
// @Summary Remove media file
// @Description Remove the file attached to a media item
// @Tags Core/Media
// @Produce json
// @Param id path int true "Media Id"
// @Success 200 {object} MediaResponse
// @Router /media/{id}/file [delete]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) RemoveFile(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id parameter"})
	}

	item, err := c.Service.RemoveFile(ctx, uint(id))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// Update godoc
// @Summary Update a media item
// @Description Update a media item's details and optionally its file
// @Tags Core/Media
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Media Id"
// @Param name formData string false "Media name"
// @Param type formData string false "Media type"
// @Param description formData string false "Media description"
// @Param file formData file false "Media file"
// @Success 200 {object} MediaResponse
// @Router /media/{id} [put]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) Update(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id parameter"})
	}

	var req UpdateMediaRequest
	if err := ctx.ShouldBind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	// Handle file upload
	if file, err := ctx.FormFile("file"); err == nil {
		req.File = file
	}

	item, err := c.Service.Update(uint(id), &req)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// Delete godoc
// @Summary Delete a media item
// @Description Delete a media item and its associated file
// @Tags Core/Media
// @Produce json
// @Param id path int true "Media Id"
// @Success 204 "No Content"
// @Router /media/{id} [delete]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) Delete(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id parameter"})
	}

	if err := c.Service.Delete(uint(id)); err != nil {
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	ctx.Status(http.StatusNoContent)
	return nil
}

// Get godoc
// @Summary Get a media item
// @Description Get a media item by Id
// @Tags Core/Media
// @Produce json
// @Param id path int true "Media Id"
// @Success 200 {object} MediaResponse
// @Router /media/{id} [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) Get(ctx *router.Context) error {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id parameter"})
	}

	item, err := c.Service.GetById(uint(id))
	if err != nil {
		return ctx.JSON(http.StatusNotFound, ErrorResponse{Error: "media not found"})
	}

	return ctx.JSON(http.StatusOK, item.ToResponse())
}

// List godoc
// @Summary List media items
// @Description Get a paginated list of media items with filtering support
// @Tags Core/Media
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param parent_id query int false "Parent folder ID for hierarchical navigation"
// @Param folder query string false "Folder path for filtering"
// @Param type query string false "Media type for filtering (e.g., image, audio, video)"
// @Success 200 {object} types.PaginatedResponse
// @Router /media [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) List(ctx *router.Context) error {
	page := 1
	limit := 10

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse filtering parameters
	filters := &MediaFilters{}

	// Parse parent_id parameter
	if parentIdStr := ctx.Query("parent_id"); parentIdStr != "" {
		if parentId, err := strconv.ParseUint(parentIdStr, 10, 32); err == nil {
			parentIdUint := uint(parentId)
			filters.ParentId = &parentIdUint
		}
	}

	// Parse folder parameter
	if folderStr := ctx.Query("folder"); folderStr != "" {
		filters.Folder = folderStr
	}

	// Parse type parameter
	if typeStr := ctx.Query("type"); typeStr != "" {
		filters.Type = typeStr
	}

	// Get author ID from context or header
	var authorId uint
	if aid, exists := ctx.Get("author_id"); exists {
		if authorIdUint, ok := aid.(uint); ok {
			authorId = authorIdUint
		}
	} else if authorIdStr := ctx.GetHeader("Base-Author-Id"); authorIdStr != "" {
		if aid, err := strconv.ParseUint(authorIdStr, 10, 32); err == nil {
			authorId = uint(aid)
		}
	}

	// Filter by author ID if available
	if authorId > 0 {
		filters.AuthorId = &authorId
		// Include shared files (author_id = null) when filtering by author
		filters.IncludeShared = true
	}

	// Use filtering method instead of basic GetAll
	result, err := c.Service.GetAllWithFilters(&page, &limit, filters)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListAll godoc
// @Summary List all media items
// @Description Get an unpaginated list of all media items with filtering support
// @Tags Core/Media
// @Produce json
// @Param parent_id query int false "Parent folder ID for hierarchical navigation"
// @Param folder query string false "Folder path for filtering"
// @Param type query string false "Media type for filtering (e.g., image, audio, video)"
// @Success 200 {array} MediaListResponse
// @Router /media/all [get]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) ListAll(ctx *router.Context) error {
	// Parse filtering parameters
	filters := &MediaFilters{}

	// Parse parent_id parameter
	if parentIdStr := ctx.Query("parent_id"); parentIdStr != "" {
		if parentId, err := strconv.ParseUint(parentIdStr, 10, 32); err == nil {
			parentIdUint := uint(parentId)
			filters.ParentId = &parentIdUint
		}
	}

	// Parse folder parameter
	if folderStr := ctx.Query("folder"); folderStr != "" {
		filters.Folder = folderStr
	}

	// Parse type parameter
	if typeStr := ctx.Query("type"); typeStr != "" {
		filters.Type = typeStr
	}

	// Use filtering method without pagination
	result, err := c.Service.GetAllWithFilters(nil, nil, filters)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, result)
}

// SyncFromR2 godoc
// @Summary Sync media from R2 bucket
// @Description Sync all files from R2 bucket to media database
// @Tags Core/Media
// @Accept json
// @Produce json
// @Param body body object false "Sync options" example({"prefix": "media/"})
// @Success 200 {object} SyncResult
// @Router /media/sync [post]
// @Security ApiKeyAuth
// @Security BearerAuth
func (c *MediaController) SyncFromR2(ctx *router.Context) error {
	// Parse request body for options
	var req struct {
		Prefix string `json:"prefix"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		req.Prefix = "media/" // Default prefix
	}

	// Get R2 config from environment
	bucket := os.Getenv("STORAGE_BUCKET")
	cdnURL := os.Getenv("CDN")
	if cdnURL == "" {
		cdnURL = os.Getenv("STORAGE_PUBLIC_URL")
	}

	// Run sync
	result, err := c.Service.SyncFromR2(c.Storage, bucket, cdnURL, req.Prefix)
	if err != nil {
		c.Logger.Error("Failed to sync from R2", logger.String("error", err.Error()))
		return ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	c.Logger.Info("R2 sync completed",
		logger.Int("total", result.TotalFiles),
		logger.Int("processed", result.ProcessedFiles),
		logger.Int("skipped", result.SkippedFiles),
		logger.Int("failed", result.FailedFiles),
	)

	return ctx.JSON(http.StatusOK, result)
}

type ErrorResponse struct {
	Error string `json:"error"`
}
