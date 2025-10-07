package storage

import (
	"path/filepath"
	"strings"
)

// MediaType represents the type of media file
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
	MediaTypeOther    MediaType = "other"
)

// DetectMediaType detects the media type from filename extension
func DetectMediaType(filename string) MediaType {
	ext := strings.ToLower(filepath.Ext(filename))

	// Image formats
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif", ".svg"}
	for _, e := range imageExts {
		if ext == e {
			return MediaTypeImage
		}
	}

	// Video formats
	videoExts := []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv", ".m4v", ".mpeg", ".mpg"}
	for _, e := range videoExts {
		if ext == e {
			return MediaTypeVideo
		}
	}

	// Audio formats
	audioExts := []string{".mp3", ".wav", ".flac", ".aac", ".m4a", ".ogg", ".wma", ".opus"}
	for _, e := range audioExts {
		if ext == e {
			return MediaTypeAudio
		}
	}

	// Document formats
	docExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv"}
	for _, e := range docExts {
		if ext == e {
			return MediaTypeDocument
		}
	}

	return MediaTypeOther
}

// GetTargetFormat returns the target format for conversion based on media type
func GetTargetFormat(mediaType MediaType) string {
	switch mediaType {
	case MediaTypeImage:
		return "webp"
	case MediaTypeVideo:
		return "webm"
	case MediaTypeAudio:
		return "opus"
	default:
		return "" // No conversion
	}
}

// ShouldConvert checks if a file should be converted based on its current format
func ShouldConvert(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	mediaType := DetectMediaType(filename)
	targetFormat := GetTargetFormat(mediaType)

	if targetFormat == "" {
		return false // No target format, don't convert
	}

	// Don't convert if already in target format
	return ext != "."+targetFormat
}
