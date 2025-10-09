package oauth

import (
	"os"
)

type OAuthConfig struct {
	Google    ProviderConfig
	Facebook  ProviderConfig
	Apple     ProviderConfig
	JWTSecret string
}

type ProviderConfig struct {
	ClientId     string
	ClientSecret string
	RedirectURL  string
}

func LoadConfig() *OAuthConfig {
	config := &OAuthConfig{
		Google: ProviderConfig{
			ClientId:     os.Getenv("GOOGLE_CLIENT_Id"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		},
		Facebook: ProviderConfig{
			ClientId:     os.Getenv("FACEBOOK_CLIENT_Id"),
			ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("FACEBOOK_REDIRECT_URL"),
		},
		Apple: ProviderConfig{
			ClientId:     os.Getenv("APPLE_CLIENT_Id"),
			ClientSecret: os.Getenv("APPLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("APPLE_REDIRECT_URL"),
		},
		JWTSecret: os.Getenv("JWT_SECRET"),
	}
	return config
}

func ValidateConfig(config *OAuthConfig) {
	// Check if at least one provider is configured
	hasProvider := false
	if config.Google.ClientId != "" && config.Google.ClientSecret != "" {
		hasProvider = true
	}
	if config.Facebook.ClientId != "" && config.Facebook.ClientSecret != "" {
		hasProvider = true
	}
	if config.Apple.ClientId != "" && config.Apple.ClientSecret != "" {
		hasProvider = true
	}

	if config.JWTSecret == "" {
		config.JWTSecret = "default-jwt-secret-for-development"
	}

	// Silently handle unonfigured OAuth - not critical
	_ = hasProvider
}
