package storage

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// VideoConverter handles video conversion operations
type VideoConverter struct {
	Quality int // CRF quality (0-51, lower is better)
}

// NewVideoConverter creates a new video converter
func NewVideoConverter(quality int) *VideoConverter {
	if quality < 0 || quality > 51 {
		quality = 23 // Default quality (good balance)
	}
	return &VideoConverter{
		Quality: quality,
	}
}

// IsVideoFile checks if the file is a supported video type
func (vc *VideoConverter) IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".mp4", ".mov", ".avi", ".mkv", ".flv", ".wmv", ".webm", ".m4v", ".mpeg", ".mpg"}
	for _, supported := range supportedExts {
		if ext == supported {
			return true
		}
	}
	return false
}

// ConvertToWebM converts a video file to WebM format and returns the bytes and new filename
func (vc *VideoConverter) ConvertToWebM(file *multipart.FileHeader) ([]byte, string, error) {
	// Check if it's a video file
	if !vc.IsVideoFile(file.Filename) {
		return nil, file.Filename, nil // Return nil if not a video (will use original)
	}

	// Check if already WebM
	if strings.ToLower(filepath.Ext(file.Filename)) == ".webm" {
		return nil, file.Filename, nil // Return nil if already WebM (will use original)
	}

	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, "", fmt.Errorf("ffmpeg not found: video conversion requires ffmpeg to be installed")
	}

	// Create temp file for input
	tmpInput, err := os.CreateTemp("", "video-input-*"+filepath.Ext(file.Filename))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(tmpInput.Name())
	defer tmpInput.Close()

	// Write uploaded file to temp location
	src, err := file.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	if _, err := tmpInput.ReadFrom(src); err != nil {
		return nil, "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpInput.Close()

	// Create temp file for output
	tmpOutput, err := os.CreateTemp("", "video-output-*.webm")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp output file: %w", err)
	}
	defer os.Remove(tmpOutput.Name())
	tmpOutput.Close()

	// Convert to WebM using ffmpeg with timeout
	// -crf: Constant Rate Factor for quality (0-63, lower is better)
	// -b:v 0: Tell VP9 to use constant quality mode
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", tmpInput.Name(),
		"-c:v", "libvpx-vp9", // VP9 codec
		"-crf", fmt.Sprintf("%d", vc.Quality),
		"-b:v", "0",
		"-c:a", "libopus", // Opus audio codec
		"-y", // Overwrite output file
		tmpOutput.Name(),
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("ffmpeg conversion failed: %w, stderr: %s", err, stderr.String())
	}

	// Read converted file
	data, err := os.ReadFile(tmpOutput.Name())
	if err != nil {
		return nil, "", fmt.Errorf("failed to read converted file: %w", err)
	}

	// Create new filename with .webm extension
	ext := filepath.Ext(file.Filename)
	newFilename := strings.TrimSuffix(file.Filename, ext) + ".webm"

	return data, newFilename, nil
}
