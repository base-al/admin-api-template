package media

import (
	"mime/multipart"
	"time"

	"base/core/storage"

	"gorm.io/gorm"
)

// Media represents a media entity
type Media struct {
	Id          uint                `json:"id" gorm:"primaryKey"`
	Name        string              `json:"name" gorm:"column:name"`
	Type        string              `json:"type" gorm:"column:type"`
	Description string              `json:"description" gorm:"column:description"`
	ParentId    *uint               `json:"parent_id" gorm:"column:parent_id;index"`   // Reference to parent folder
	Folder      string              `json:"folder" gorm:"column:folder;index"`         // Computed full path for compatibility
	Tags        string              `json:"tags" gorm:"column:tags"`                   // Comma-separated tags for searching
	Metadata    *string             `json:"metadata" gorm:"column:metadata;type:json"` // JSON metadata for extra properties (nullable)
	AuthorId    *uint               `json:"author_id" gorm:"column:author_id;index"`   // Optional author ownership
	File        *storage.Attachment `json:"file,omitempty" gorm:"polymorphic:Model"`

	// Conversion tracking
	OriginalFormat  string `json:"original_format,omitempty" gorm:"column:original_format"`   // Format before conversion (mp4, png, mp3)
	ConvertedFormat string `json:"converted_format,omitempty" gorm:"column:converted_format"` // Format after conversion (webm, webp, opus)

	// Relationships
	Parent   *Media   `json:"parent,omitempty" gorm:"foreignKey:ParentId"`
	Children []*Media `json:"children,omitempty" gorm:"foreignKey:ParentId"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName returns the table name for the Media model
func (item *Media) TableName() string {
	return "media"
}

// GetId returns the Id of the model
func (item *Media) GetId() uint {
	return item.Id
}

// GetModelName returns the model name
func (item *Media) GetModelName() string {
	return "media"
}

// Preload preloads all the model's relationships
func (item *Media) Preload(db *gorm.DB) *gorm.DB {
	return db.Preload("File").Preload("Parent").Preload("Children")
}

// MediaListResponse represents the list view response
type MediaListResponse struct {
	Id          uint                `json:"id"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	Description string              `json:"description"`
	ParentId    *uint               `json:"parent_id"`
	Folder      string              `json:"folder"`
	Tags        string              `json:"tags"`
	AuthorId    *uint               `json:"author_id"`
	File        *storage.Attachment `json:"file,omitempty"`
}

// MediaResponse represents the detailed view response
type MediaResponse struct {
	Id          uint                `json:"id"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   gorm.DeletedAt      `json:"deleted_at,omitempty"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	Description string              `json:"description"`
	ParentId    *uint               `json:"parent_id"`
	Folder      string              `json:"folder"`
	Tags        string              `json:"tags"`
	Metadata    *string             `json:"metadata"`
	AuthorId    *uint               `json:"author_id"`
	File        *storage.Attachment `json:"file,omitempty"`
	Parent      *Media              `json:"parent,omitempty"`
	Children    []*Media            `json:"children,omitempty"`
}

// MediaResponse represents the detailed view response
type MediaModelResponse struct {
	Id          uint                `json:"id"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   gorm.DeletedAt      `json:"deleted_at,omitempty"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	Description string              `json:"description"`
	File        *storage.Attachment `json:"file,omitempty"`
}

// CreateMediaRequest represents the request payload for creating a Media
type CreateMediaRequest struct {
	Name        string                `form:"name" json:"name" binding:"required"`
	Type        string                `form:"type" json:"type" binding:"required"`
	Description string                `form:"description" json:"description"`
	ParentId    *uint                 `json:"parent_id"`                      // For JSON requests
	Folder      string                `form:"folder" json:"folder"`           // Optional folder path (for compatibility)
	Tags        string                `form:"tags" json:"tags"`               // Optional comma-separated tags
	Metadata    string                `form:"metadata" json:"metadata"`       // Optional JSON metadata
	AuthorId    *uint                 `json:"author_id"`                      // For JSON requests
	File        *multipart.FileHeader `form:"file"`
}

// UpdateMediaRequest represents the request payload for updating a Media
type UpdateMediaRequest struct {
	Name        *string               `form:"name"`
	Type        *string               `form:"type"`
	Description *string               `form:"description"`
	ParentId    *uint                 `form:"parent_id"`
	Folder      *string               `form:"folder"`
	Tags        *string               `form:"tags"`
	Metadata    *string               `form:"metadata"`
	AuthorId    *uint                 `form:"author_id"`
	File        *multipart.FileHeader `form:"file"`
}

// ToListResponse converts the model to a list response
func (item *Media) ToListResponse() *MediaListResponse {
	return &MediaListResponse{
		Id:          item.Id,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		Name:        item.Name,
		Type:        item.Type,
		Description: item.Description,
		ParentId:    item.ParentId,
		Folder:      item.Folder,
		Tags:        item.Tags,
		AuthorId:    item.AuthorId,
		File:        item.File,
	}
}

// ToResponse converts the model to a detailed response
func (item *Media) ToResponse() *MediaResponse {
	return &MediaResponse{
		Id:          item.Id,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		DeletedAt:   item.DeletedAt,
		Name:        item.Name,
		Type:        item.Type,
		Description: item.Description,
		ParentId:    item.ParentId,
		Folder:      item.Folder,
		Tags:        item.Tags,
		Metadata:    item.Metadata,
		AuthorId:    item.AuthorId,
		File:        item.File,
		Parent:      item.Parent,
		Children:    item.Children,
	}
}

// ToResponse converts the model to a detailed response
func (item *Media) ToModelResponse() *MediaModelResponse {
	return &MediaModelResponse{
		Id:          item.Id,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		DeletedAt:   item.DeletedAt,
		Name:        item.Name,
		Type:        item.Type,
		Description: item.Description,
		File:        item.File,
	}
}

var _ storage.Attachable = (*Media)(nil)

// GetAttachmentConfig returns the attachment configuration for the model
func (item *Media) GetAttachmentConfig() map[string]interface{} {
	return map[string]interface{}{
		"file": map[string]interface{}{
			"path":       "media/:id/:filename",
			"validators": []string{"image", "audio"},
			"min_size":   1,                 // 1 byte
			"max_size":   100 * 1024 * 1024, // 100MB
		},
	}
}

// Helper methods for tree operations

// GetPath returns the full path from root to this item
func (item *Media) GetPath() string {
	if item.Parent == nil {
		return item.Name
	}
	return item.Parent.GetPath() + "/" + item.Name
}

// UpdateFolderPath computes and updates the folder path based on parent relationship
func (item *Media) UpdateFolderPath(db *gorm.DB) error {
	if item.ParentId == nil {
		item.Folder = ""
	} else {
		var parent Media
		if err := db.First(&parent, *item.ParentId).Error; err != nil {
			return err
		}
		if parent.ParentId == nil {
			item.Folder = parent.Name
		} else {
			parentPath := parent.GetPath()
			item.Folder = parentPath
		}
	}
	return nil
}

// IsDescendantOf checks if this item is a descendant of another item
func (item *Media) IsDescendantOf(ancestor *Media) bool {
	if item.ParentId == nil {
		return false
	}
	if *item.ParentId == ancestor.Id {
		return true
	}
	if item.Parent != nil {
		return item.Parent.IsDescendantOf(ancestor)
	}
	return false
}

// GetDepth returns the depth of this item in the tree (root = 0)
func (item *Media) GetDepth() int {
	if item.Parent == nil {
		return 0
	}
	return item.Parent.GetDepth() + 1
}

// MediaFilters represents filtering options for media queries
type MediaFilters struct {
	ParentId      *uint  `json:"parent_id"`
	Folder        string `json:"folder"`
	Type          string `json:"type"`
	AuthorId      *uint  `json:"author_id"`
	IncludeShared bool   `json:"include_shared"`
}
