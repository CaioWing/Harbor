package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Server  ServerConfig
	DB      DBConfig
	Auth    AuthConfig
	Storage StorageConfig
	CORS    CORSConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

type AuthConfig struct {
	JWTSecret          string
	JWTExpiry          time.Duration
	DeviceTokenExpiry  time.Duration
}

type StorageConfig struct {
	Path string
}

type CORSConfig struct {
	AllowedOrigins string
}

func Load() (*Config, error) {
	jwtExpiry, err := time.ParseDuration(envOrDefault("HARBOR_JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid HARBOR_JWT_EXPIRY: %w", err)
	}

	deviceTokenExpiry, err := time.ParseDuration(envOrDefault("HARBOR_DEVICE_TOKEN_EXPIRY", "8760h"))
	if err != nil {
		return nil, fmt.Errorf("invalid HARBOR_DEVICE_TOKEN_EXPIRY: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: envOrDefault("HARBOR_HOST", "0.0.0.0"),
			Port: envOrDefault("HARBOR_PORT", "8080"),
		},
		DB: DBConfig{
			Host:     envOrDefault("HARBOR_DB_HOST", "localhost"),
			Port:     envOrDefault("HARBOR_DB_PORT", "5432"),
			Name:     envOrDefault("HARBOR_DB_NAME", "harbor"),
			User:     envOrDefault("HARBOR_DB_USER", "harbor"),
			Password: envOrDefault("HARBOR_DB_PASSWORD", "harbor"),
			SSLMode:  envOrDefault("HARBOR_DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			JWTSecret:         envOrDefault("HARBOR_JWT_SECRET", "change-me-in-production"),
			JWTExpiry:         jwtExpiry,
			DeviceTokenExpiry: deviceTokenExpiry,
		},
		Storage: StorageConfig{
			Path: envOrDefault("HARBOR_STORAGE_PATH", "/data/artifacts"),
		},
		CORS: CORSConfig{
			AllowedOrigins: envOrDefault("HARBOR_CORS_ORIGINS", "http://localhost:3000"),
		},
	}

	return cfg, nil
}

func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
