package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv         string
	AppBaseURL     string
	HTTPPort       string
	UserWebOrigin  string
	AdminWebOrigin string

	DatabaseURL string
	RedisURL    string

	SessionSecret string
	CSRFSecret    string
	LogLevel      string
	AutoMigrate   bool
}

// Load loads configuration from environment variables, optionally reading from an .env file.
func Load() *Config {
	// Try loading .env file if it exists
	loadDotEnv(".env")
	loadDotEnv("../.env")

	cfg := &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		AppBaseURL:     getEnv("APP_BASE_URL", "http://localhost:8080"),
		HTTPPort:       getEnv("HTTP_PORT", "8080"),
		UserWebOrigin:  getEnv("USER_WEB_ORIGIN", "http://localhost:3000"),
		AdminWebOrigin: getEnv("ADMIN_WEB_ORIGIN", "http://localhost:3001"),

		DatabaseURL: getEnv("DATABASE_URL", "postgres://vps_billing:change-me@localhost:5432/vps_billing?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),

		SessionSecret: getEnv("SESSION_SECRET", "default-insecure-dev-session-secret-min-32-bytes"),
		CSRFSecret:    getEnv("CSRF_SECRET", "default-insecure-dev-csrf-secret-min-32-bytes"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		AutoMigrate:   getEnvBool("AUTO_MIGRATE", true),
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
	}
}
