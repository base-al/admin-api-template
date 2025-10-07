package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
)

// ImageProcessor handles image conversion operations
type ImageProcessor struct {
	Quality int // WebP quality (0-100)
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(quality int) *ImageProcessor {
	if quality <= 0 || quality > 100 {
		quality = 85 // Default quality
	}
	return &ImageProcessor{
		Quality: quality,
	}
}

// IsImageFile checks if the file is a supported image type
func (ip *ImageProcessor) IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".jpg", ".jpeg", ".png", ".bmp", ".tiff", ".tif", ".webp"}
	for _, supported := range supportedExts {
		if ext == supported {
			return true
		}
	}
	return false
}

// ConvertToWebP converts an image file to WebP format and returns the bytes and new filename
func (ip *ImageProcessor) ConvertToWebP(file *multipart.FileHeader) ([]byte, string, error) {
	// Check if it's an image file
	if !ip.IsImageFile(file.Filename) {
		return nil, file.Filename, nil // Return nil bytes if not an image (will use original)
	}

	// Check if already WebP
	if strings.ToLower(filepath.Ext(file.Filename)) == ".webp" {
		return nil, file.Filename, nil // Return nil bytes if already WebP (will use original)
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read all file data
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	// Decode the image
	img, err := ip.decodeImage(bytes.NewReader(data), file.Filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Encode to WebP
	var buf bytes.Buffer
	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, float32(ip.Quality))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create encoder options: %w", err)
	}

	if err := webp.Encode(&buf, img, options); err != nil {
		return nil, "", fmt.Errorf("failed to encode to webp: %w", err)
	}

	// Create new filename with .webp extension
	ext := filepath.Ext(file.Filename)
	newFilename := strings.TrimSuffix(file.Filename, ext) + ".webp"

	return buf.Bytes(), newFilename, nil
}

// decodeImage decodes an image from a reader based on file extension
func (ip *ImageProcessor) decodeImage(r io.Reader, filename string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(r)
	case ".png":
		return png.Decode(r)
	case ".webp":
		return webp.Decode(r, nil)
	default:
		// Try generic decode
		img, _, err := image.Decode(r)
		return img, err
	}
}

// ConvertToWebPBytes converts image bytes to WebP format
func (ip *ImageProcessor) ConvertToWebPBytes(data []byte, originalFilename string) ([]byte, string, error) {
	// Check if it's an image file
	if !ip.IsImageFile(originalFilename) {
		return data, originalFilename, nil
	}

	// Check if already WebP
	if strings.ToLower(filepath.Ext(originalFilename)) == ".webp" {
		return data, originalFilename, nil
	}

	// Decode the image
	img, err := ip.decodeImage(bytes.NewReader(data), originalFilename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Encode to WebP
	var buf bytes.Buffer
	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, float32(ip.Quality))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create encoder options: %w", err)
	}

	if err := webp.Encode(&buf, img, options); err != nil {
		return nil, "", fmt.Errorf("failed to encode to webp: %w", err)
	}

	// Create new filename
	ext := filepath.Ext(originalFilename)
	newFilename := strings.TrimSuffix(originalFilename, ext) + ".webp"

	return buf.Bytes(), newFilename, nil
}
