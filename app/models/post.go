package models

import (
	"base/core/storage"
	"base/core/types"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Post represents a post entity
type Post struct {
	Id            uint                `json:"id" gorm:"primarykey"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	DeletedAt     gorm.DeletedAt      `json:"deleted_at" gorm:"index"`
	Title         string              `json:"title"`
	Slug          string              `json:"slug"`
	Content       string              `json:"content"`
	Excerpt       string              `json:"excerpt"`
	AuthorId      uint                `json:"author_id"`
	Status        string              `json:"status"`
	Category      string              `json:"category"`
	Published     bool                `json:"published"`
	Featured      bool                `json:"featured"`
	IsPinned      bool                `json:"is_pinned"`
	ViewCount     int                 `json:"view_count"`
	LikeCount     int                 `json:"like_count"`
	Rating        float64             `json:"rating"`
	PublishedAt   types.DateTime      `json:"published_at"`
	ScheduledAt   types.DateTime      `json:"scheduled_at"`
	Tags          json.RawMessage     `json:"tags"`
	Metadata      json.RawMessage     `json:"metadata"`
	FeaturedImage *storage.Attachment `json:"featured_image,omitempty" gorm:"foreignKey:ModelId;references:Id"`
}

// TableName returns the table name for the Post model
func (m *Post) TableName() string {
	return "posts"
}

// GetId returns the Id of the model
func (m *Post) GetId() uint {
	return m.Id
}

// GetModelName returns the model name
func (m *Post) GetModelName() string {
	return "post"
}

// CreatePostRequest represents the request payload for creating a Post
type CreatePostRequest struct {
	Title       string          `json:"title"`
	Slug        string          `json:"slug"`
	Content     string          `json:"content"`
	Excerpt     string          `json:"excerpt"`
	AuthorId    uint            `json:"author_id"`
	Status      string          `json:"status"`
	Category    string          `json:"category"`
	Published   bool            `json:"published"`
	Featured    bool            `json:"featured"`
	IsPinned    bool            `json:"is_pinned"`
	ViewCount   int             `json:"view_count"`
	LikeCount   int             `json:"like_count"`
	Rating      float64         `json:"rating"`
	PublishedAt types.DateTime  `json:"published_at" swaggertype:"string"`
	ScheduledAt types.DateTime  `json:"scheduled_at" swaggertype:"string"`
	Tags        json.RawMessage `json:"tags"`
	Metadata    json.RawMessage `json:"metadata"`
}

// UpdatePostRequest represents the request payload for updating a Post
type UpdatePostRequest struct {
	Title       string          `json:"title,omitempty"`
	Slug        string          `json:"slug,omitempty"`
	Content     string          `json:"content,omitempty"`
	Excerpt     string          `json:"excerpt,omitempty"`
	AuthorId    uint            `json:"author_id,omitempty"`
	Status      string          `json:"status,omitempty"`
	Category    string          `json:"category,omitempty"`
	Published   *bool           `json:"published,omitempty"`
	Featured    *bool           `json:"featured,omitempty"`
	IsPinned    *bool           `json:"is_pinned,omitempty"`
	ViewCount   int             `json:"view_count,omitempty"`
	LikeCount   int             `json:"like_count,omitempty"`
	Rating      float64         `json:"rating,omitempty"`
	PublishedAt types.DateTime  `json:"published_at,omitempty" swaggertype:"string"`
	ScheduledAt types.DateTime  `json:"scheduled_at,omitempty" swaggertype:"string"`
	Tags        json.RawMessage `json:"tags,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// PostResponse represents the API response for Post
type PostResponse struct {
	Id            uint                `json:"id"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	DeletedAt     gorm.DeletedAt      `json:"deleted_at"`
	Title         string              `json:"title"`
	Slug          string              `json:"slug"`
	Content       string              `json:"content"`
	Excerpt       string              `json:"excerpt"`
	AuthorId      uint                `json:"author_id"`
	Status        string              `json:"status"`
	Category      string              `json:"category"`
	Published     bool                `json:"published"`
	Featured      bool                `json:"featured"`
	IsPinned      bool                `json:"is_pinned"`
	ViewCount     int                 `json:"view_count"`
	LikeCount     int                 `json:"like_count"`
	Rating        float64             `json:"rating"`
	PublishedAt   types.DateTime      `json:"published_at"`
	ScheduledAt   types.DateTime      `json:"scheduled_at"`
	Tags          json.RawMessage     `json:"tags"`
	Metadata      json.RawMessage     `json:"metadata"`
	FeaturedImage *storage.Attachment `json:"featured_image,omitempty"`
}

// PostModelResponse represents a simplified response when this model is part of other entities
type PostModelResponse struct {
	Id    uint   `json:"id"`
	Title string `json:"title"`
}

// PostSelectOption represents a simplified response for select boxes and dropdowns
type PostSelectOption struct {
	Id   uint   `json:"id"`
	Name string `json:"name"` // From Title field
}

// PostListResponse represents the response for list operations (optimized for performance)
type PostListResponse struct {
	Id            uint                `json:"id"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	DeletedAt     gorm.DeletedAt      `json:"deleted_at"`
	Title         string              `json:"title"`
	Slug          string              `json:"slug"`
	Content       string              `json:"content"`
	Excerpt       string              `json:"excerpt"`
	AuthorId      uint                `json:"author_id"`
	Status        string              `json:"status"`
	Category      string              `json:"category"`
	Published     bool                `json:"published"`
	Featured      bool                `json:"featured"`
	IsPinned      bool                `json:"is_pinned"`
	ViewCount     int                 `json:"view_count"`
	LikeCount     int                 `json:"like_count"`
	Rating        float64             `json:"rating"`
	PublishedAt   types.DateTime      `json:"published_at"`
	ScheduledAt   types.DateTime      `json:"scheduled_at"`
	Tags          json.RawMessage     `json:"tags"`
	Metadata      json.RawMessage     `json:"metadata"`
	FeaturedImage *storage.Attachment `json:"featured_image,omitempty"`
}

// ToResponse converts the model to an API response
func (m *Post) ToResponse() *PostResponse {
	if m == nil {
		return nil
	}
	response := &PostResponse{
		Id:          m.Id,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
		Title:       m.Title,
		Slug:        m.Slug,
		Content:     m.Content,
		Excerpt:     m.Excerpt,
		AuthorId:    m.AuthorId,
		Status:      m.Status,
		Category:    m.Category,
		Published:   m.Published,
		Featured:    m.Featured,
		IsPinned:    m.IsPinned,
		ViewCount:   m.ViewCount,
		LikeCount:   m.LikeCount,
		Rating:      m.Rating,
		PublishedAt: m.PublishedAt,
		ScheduledAt: m.ScheduledAt,
		Tags:        m.Tags,
		Metadata:    m.Metadata,
	}
	if m.FeaturedImage != nil {
		response.FeaturedImage = m.FeaturedImage
	}

	return response
}

// ToModelResponse converts the model to a simplified response for when it's part of other entities
func (m *Post) ToModelResponse() *PostModelResponse {
	if m == nil {
		return nil
	}
	return &PostModelResponse{
		Id:    m.Id,
		Title: m.Title,
	}
}

// ToSelectOption converts the model to a select option for dropdowns
func (m *Post) ToSelectOption() *PostSelectOption {
	if m == nil {
		return nil
	}
	displayName := m.Title

	return &PostSelectOption{
		Id:   m.Id,
		Name: displayName,
	}
}

// ToListResponse converts the model to a list response (without preloaded relationships for fast listing)
func (m *Post) ToListResponse() *PostListResponse {
	if m == nil {
		return nil
	}
	return &PostListResponse{
		Id:          m.Id,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
		Title:       m.Title,
		Slug:        m.Slug,
		Content:     m.Content,
		Excerpt:     m.Excerpt,
		AuthorId:    m.AuthorId,
		Status:      m.Status,
		Category:    m.Category,
		Published:   m.Published,
		Featured:    m.Featured,
		IsPinned:    m.IsPinned,
		ViewCount:   m.ViewCount,
		LikeCount:   m.LikeCount,
		Rating:      m.Rating,
		PublishedAt: m.PublishedAt,
		ScheduledAt: m.ScheduledAt,
		Tags:        m.Tags,
		Metadata:    m.Metadata,
	}
}

// Preload preloads all the model's relationships
func (m *Post) Preload(db *gorm.DB) *gorm.DB {
	query := db
	return query
}
