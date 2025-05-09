package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// DBConfig holds database configuration
type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Config holds all configuration for the application
type Config struct {
	FusionBrainAPIKey     string
	FusionBrainSecretKey  string
	DefaultImageWidth     int
	DefaultImageHeight    int
	DefaultNumImages      int
	DefaultStyle          string
	DefaultNegativePrompt string
	GenerationTimeout     time.Duration
	CheckInterval         time.Duration
	MaxAttempts           int
	DB                    DBConfig
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	config := &Config{
		FusionBrainAPIKey:    os.Getenv("FUSION_BRAIN_API_KEY"),
		FusionBrainSecretKey: os.Getenv("FUSION_BRAIN_SECRET_KEY"),
		DefaultStyle:         os.Getenv("DEFAULT_STYLE"),
	}

	// Load and parse numeric values
	if width, err := strconv.Atoi(os.Getenv("DEFAULT_IMAGE_WIDTH")); err == nil {
		config.DefaultImageWidth = width
	} else {
		config.DefaultImageWidth = 1024 // default value
	}

	if height, err := strconv.Atoi(os.Getenv("DEFAULT_IMAGE_HEIGHT")); err == nil {
		config.DefaultImageHeight = height
	} else {
		config.DefaultImageHeight = 1024 // default value
	}

	if numImages, err := strconv.Atoi(os.Getenv("DEFAULT_NUM_IMAGES")); err == nil {
		config.DefaultNumImages = numImages
	} else {
		config.DefaultNumImages = 1 // default value
	}

	if timeout, err := strconv.Atoi(os.Getenv("DEFAULT_GENERATION_TIMEOUT")); err == nil {
		config.GenerationTimeout = time.Duration(timeout) * time.Second
	} else {
		config.GenerationTimeout = 5 * time.Minute // default value
	}

	if interval, err := strconv.Atoi(os.Getenv("DEFAULT_CHECK_INTERVAL")); err == nil {
		config.CheckInterval = time.Duration(interval) * time.Second
	} else {
		config.CheckInterval = 2 * time.Second // default value
	}

	if attempts, err := strconv.Atoi(os.Getenv("DEFAULT_MAX_ATTEMPTS")); err == nil {
		config.MaxAttempts = attempts
	} else {
		config.MaxAttempts = 30 // default value
	}

	// Load database configuration
	dbConfig := DBConfig{
		Host:     os.Getenv("DB_HOST"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSL_MODE"),
	}

	// Parse database port
	if port, err := strconv.Atoi(os.Getenv("DB_PORT")); err == nil {
		dbConfig.Port = port
	} else {
		dbConfig.Port = 5432 // default PostgreSQL port
	}

	// Parse connection pool settings
	if maxOpenConns, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNS")); err == nil {
		dbConfig.MaxOpenConns = maxOpenConns
	} else {
		dbConfig.MaxOpenConns = 25 // default value
	}

	if maxIdleConns, err := strconv.Atoi(os.Getenv("DB_MAX_IDLE_CONNS")); err == nil {
		dbConfig.MaxIdleConns = maxIdleConns
	} else {
		dbConfig.MaxIdleConns = 25 // default value
	}

	if connMaxLifetime, err := strconv.Atoi(os.Getenv("DB_CONN_MAX_LIFETIME")); err == nil {
		dbConfig.ConnMaxLifetime = time.Duration(connMaxLifetime) * time.Second
	} else {
		dbConfig.ConnMaxLifetime = 5 * time.Minute // default value
	}

	config.DB = dbConfig

	// Validate required fields
	if config.FusionBrainAPIKey == "" {
		return nil, fmt.Errorf("FUSION_BRAIN_API_KEY is required")
	}
	if config.FusionBrainSecretKey == "" {
		return nil, fmt.Errorf("FUSION_BRAIN_SECRET_KEY is required")
	}

	// Validate database configuration
	if config.DB.Host == "" {
		return nil, fmt.Errorf("DB_HOST is required")
	}
	if config.DB.User == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	if config.DB.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if config.DB.Database == "" {
		return nil, fmt.Errorf("DB_NAME is required")
	}

	return config, nil
}

// GetDSN returns the PostgreSQL connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Database, c.DB.SSLMode)
}
