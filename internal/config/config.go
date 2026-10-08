package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv          string
	AppPort         string
	BaseURL         string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	UploadDir       string
	MaxUploadSizeMB int
}

func LoadConfig() (*Config, error) {
	// Load .env if present (ignore error in docker/production)
	_ = godotenv.Load(".env")

	maxSize, err := strconv.Atoi(getEnv("MAX_UPLOAD_SIZE_MB", "5"))
	if err != nil {
		maxSize = 5
	}

	cfg := &Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		AppPort:         getEnv("APP_PORT", "8080"),
		BaseURL:         getEnv("BASE_URL", "http://localhost:8080"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "postgres"),
		DBName:          getEnv("DB_NAME", "waste_collection_db"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		UploadDir:       getEnv("UPLOAD_DIR", "./uploads/payments"),
		MaxUploadSizeMB: maxSize,
	}

	return cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
