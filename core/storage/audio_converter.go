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

// AudioConverter handles audio conversion operations
type AudioConverter struct {
	Bitrate int // kbps (96 recommended for speech, 128 for music)
}

// NewAudioConverter creates a new audio converter
func NewAudioConverter(bitrate int) *AudioConverter {
	if bitrate <= 0 {
		bitrate = 96 // Default bitrate (good for speech)
	}
	return &AudioConverter{
		Bitrate: bitrate,
	}
}

// IsAudioFile checks if the file is a supported audio type
func (ac *AudioConverter) IsAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".mp3", ".wav", ".flac", ".aac", ".m4a", ".ogg", ".wma", ".opus"}
	for _, supported := range supportedExts {
		if ext == supported {
			return true
		}
	}
	return false
}

// ConvertToOpus converts an audio file to Opus format and returns the bytes and new filename
func (ac *AudioConverter) ConvertToOpus(file *multipart.FileHeader) ([]byte, string, error) {
	// Check if it's an audio file
	if !ac.IsAudioFile(file.Filename) {
		return nil, file.Filename, nil // Return nil if not audio (will use original)
	}

	// Check if already Opus
	if strings.ToLower(filepath.Ext(file.Filename)) == ".opus" {
		return nil, file.Filename, nil // Return nil if already Opus (will use original)
	}

	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, "", fmt.Errorf("ffmpeg not found: audio conversion requires ffmpeg to be installed")
	}

	// Create temp file for input
	tmpInput, err := os.CreateTemp("", "audio-input-*"+filepath.Ext(file.Filename))
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
	tmpOutput, err := os.CreateTemp("", "audio-output-*.opus")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp output file: %w", err)
	}
	defer os.Remove(tmpOutput.Name())
	tmpOutput.Close()

	// Convert to Opus using ffmpeg with timeout
	// Opus is the best audio codec for web with excellent quality at low bitrates
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", tmpInput.Name(),
		"-c:a", "libopus",                        // Opus audio codec
		"-b:a", fmt.Sprintf("%dk", ac.Bitrate),   // Bitrate
		"-vn",                                     // No video stream
		"-y",                                      // Overwrite output file
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

	// Create new filename with .opus extension
	ext := filepath.Ext(file.Filename)
	newFilename := strings.TrimSuffix(file.Filename, ext) + ".opus"

	return data, newFilename, nil
}
