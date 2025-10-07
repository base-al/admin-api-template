package storage

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

func NewActiveStorage(db *gorm.DB, config Config) (*ActiveStorage, error) {
	var provider Provider
	var err error

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// If path is relative, make it absolute using cwd
	storagePath := config.Path
	if !filepath.IsAbs(storagePath) {
		storagePath = filepath.Join(cwd, storagePath)
	}

	switch strings.ToLower(config.Provider) {
	case "local":
		provider, err = NewLocalProvider(LocalConfig{
			BasePath: storagePath,
			BaseURL:  config.BaseURL,
		})
	case "s3":
		provider, err = NewS3Provider(S3Config{
			APIKey:          config.APIKey,
			APISecret:       config.APISecret,
			AccessKeyID:     config.APIKey,
			AccessKeySecret: config.APISecret,
			AccountID:       config.AccountID,
			Endpoint:        config.Endpoint,
			Bucket:          config.Bucket,
			BaseURL:         config.BaseURL,
			Region:          config.Region,
		})
	case "r2":
		provider, err = NewR2Provider(R2Config{
			AccessKeyID:     config.APIKey,
			AccessKeySecret: config.APISecret,
			AccountID:       config.AccountID,
			Bucket:          config.Bucket,
			BaseURL:         config.BaseURL,
			CDN:             config.CDN,
		})
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", config.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage provider: %w", err)
	}

	as := &ActiveStorage{
		db:             db,
		provider:       provider,
		defaultPath:    storagePath,
		configs:        make(map[string]map[string]AttachmentConfig),
		imageProcessor: NewImageProcessor(85),  // 85% quality for WebP (will be overridden by settings)
		videoConverter: NewVideoConverter(23),  // CRF 23 for WebM (will be overridden by settings)
		audioConverter: NewAudioConverter(96),  // 96 kbps for audio (will be overridden by settings)
	}

	// Auto-migrate the Attachment model
	if err := db.AutoMigrate(&Attachment{}); err != nil {
		return nil, fmt.Errorf("failed to migrate attachments table: %w", err)
	}

	return as, nil
}

func (as *ActiveStorage) RegisterAttachment(modelName string, config AttachmentConfig) {
	if as.configs[modelName] == nil {
		as.configs[modelName] = make(map[string]AttachmentConfig)
	}
	as.configs[modelName][config.Field] = config
}

func (as *ActiveStorage) Attach(model Attachable, field string, file *multipart.FileHeader) (*Attachment, error) {
	// Get config for model
	config, err := as.getConfig(model.GetModelName(), field)
	if err != nil {
		return nil, err
	}

	// Validate file
	if err := as.validateFile(file, config); err != nil {
		return nil, err
	}

	// Get media conversion settings from database
	convertImages := as.getSettingBool("media_convert_images", true)
	convertVideos := as.getSettingBool("media_convert_videos", true)
	convertAudio := as.getSettingBool("media_convert_audio", true)

	// Try to convert images to WebP (if enabled)
	var convertedData []byte
	var convertedFilename string
	if convertImages && as.imageProcessor != nil {
		convertedData, convertedFilename, err = as.imageProcessor.ConvertToWebP(file)
		if err != nil {
			return nil, fmt.Errorf("failed to convert image to webp: %w", err)
		}
	}

	// If not converted to image, try video conversion to WebM (if enabled)
	if convertedData == nil && convertVideos && as.videoConverter != nil {
		convertedData, convertedFilename, err = as.videoConverter.ConvertToWebM(file)
		if err != nil {
			return nil, fmt.Errorf("failed to convert video to webm: %w", err)
		}
	}

	// If not converted to image or video, try audio conversion to Opus (if enabled)
	if convertedData == nil && convertAudio && as.audioConverter != nil {
		convertedData, convertedFilename, err = as.audioConverter.ConvertToOpus(file)
		if err != nil {
			return nil, fmt.Errorf("failed to convert audio to opus: %w", err)
		}
	}

	// Use converted file if available
	finalFile := file
	if convertedData != nil {
		// Create a new file header with converted data
		finalFile = &multipart.FileHeader{
			Filename: convertedFilename,
			Size:     int64(len(convertedData)),
			Header:   file.Header,
		}
	}

	// Create attachment record
	attachment := &Attachment{
		ModelType: model.GetModelName(),
		ModelId:   model.GetId(),
		Field:     field,
		Filename:  finalFile.Filename,
		Size:      finalFile.Size,
	}

	// Upload file using provider (with converted data if available)
	var result *UploadResult
	if convertedData != nil {
		result, err = as.provider.UploadBytes(convertedData, finalFile.Filename, UploadConfig{
			AllowedExtensions: config.AllowedExtensions,
			MaxFileSize:       config.MaxFileSize,
			UploadPath:        filepath.Join(config.Path, model.GetModelName(), field),
		})
	} else {
		result, err = as.provider.Upload(finalFile, UploadConfig{
			AllowedExtensions: config.AllowedExtensions,
			MaxFileSize:       config.MaxFileSize,
			UploadPath:        filepath.Join(config.Path, model.GetModelName(), field),
		})
	}

	if err != nil {
		return nil, err
	}

	// Update attachment with upload result
	attachment.Path = result.Path
	attachment.URL = as.provider.GetURL(result.Path)

	// Save attachment record
	if err := as.db.Create(attachment).Error; err != nil {
		// Try to delete uploaded file if record creation fails
		_ = as.provider.Delete(result.Path)
		return nil, err
	}

	return attachment, nil
}

func (as *ActiveStorage) Delete(attachment *Attachment) error {
	if err := as.provider.Delete(attachment.Path); err != nil {
		return err
	}
	return as.db.Delete(attachment).Error
}

// GetProvider returns the storage provider (for internal use)
func (as *ActiveStorage) GetProvider() Provider {
	return as.provider
}

func (as *ActiveStorage) getConfig(modelName, field string) (AttachmentConfig, error) {
	modelConfigs, ok := as.configs[modelName]
	if !ok {
		return AttachmentConfig{}, fmt.Errorf("no attachment config found for model %s", modelName)
	}

	config, ok := modelConfigs[field]
	if !ok {
		return AttachmentConfig{}, fmt.Errorf("no attachment config found for field %s in model %s", field, modelName)
	}

	return config, nil
}

func (as *ActiveStorage) LoadAttachment(model Attachable, field string) (*Attachment, error) {
	var attachment Attachment
	err := as.db.Where("model_type = ? AND model_id = ? AND field = ?",
		model.GetModelName(), model.GetId(), field).First(&attachment).Error
	if err != nil {
		return nil, err
	}

	// Ensure URL has full path with CDN
	attachment.URL = as.provider.GetURL(attachment.Path)

	return &attachment, nil
}

func (as *ActiveStorage) validateFile(file *multipart.FileHeader, config AttachmentConfig) error {
	if file.Size > config.MaxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", config.MaxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if len(config.AllowedExtensions) > 0 && !strings.Contains(strings.Join(config.AllowedExtensions, ","), ext) {
		return fmt.Errorf("file extension %s is not allowed", ext)
	}

	return nil
}

// getSettingBool retrieves a boolean setting from the database
func (as *ActiveStorage) getSettingBool(key string, defaultValue bool) bool {
	type Settings struct {
		ValueBool bool `gorm:"column:value_bool"`
	}
	var setting Settings
	if err := as.db.Table("settings").Select("value_bool").Where("setting_key = ?", key).First(&setting).Error; err != nil {
		return defaultValue
	}
	return setting.ValueBool
}

// getSettingInt retrieves an integer setting from the database
func (as *ActiveStorage) getSettingInt(key string, defaultValue int) int {
	type Settings struct {
		ValueInt int `gorm:"column:value_int"`
	}
	var setting Settings
	if err := as.db.Table("settings").Select("value_int").Where("setting_key = ?", key).First(&setting).Error; err != nil {
		return defaultValue
	}
	return setting.ValueInt
}
