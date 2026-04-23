package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv              string
	Port                string
	ReadTimeoutSeconds  int
	WriteTimeoutSeconds int
	CORSAllowedOrigins  []string

	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	PostgresSSLMode  string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret             string
	AccessTokenTTLMinutes int
	ListingsCacheTTL      int
}

func Load() Config {
	return Config{
		AppEnv:                getEnv("APP_ENV", "development"),
		Port:                  getEnv("PORT", "8080"),
		ReadTimeoutSeconds:    getEnvInt("READ_TIMEOUT_SECONDS", 15),
		WriteTimeoutSeconds:   getEnvInt("WRITE_TIMEOUT_SECONDS", 15),
		CORSAllowedOrigins:    getEnvList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		PostgresHost:          getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:          getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:            getEnv("POSTGRES_DB", "zedsellauto"),
		PostgresUser:          getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword:      getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresSSLMode:       getEnv("POSTGRES_SSLMODE", "disable"),
		RedisAddr:             getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:         getEnv("REDIS_PASSWORD", ""),
		RedisDB:               getEnvInt("REDIS_DB", 0),
		JWTSecret:             getEnv("JWT_SECRET", "change-me"),
		AccessTokenTTLMinutes: getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 120),
		ListingsCacheTTL:      getEnvInt("LISTINGS_CACHE_TTL_SECONDS", 60),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	if len(result) == 0 {
		return fallback
	}

	return result
}

func LoadDotEnv(path string) {
	file, err := os.Open(filepath.Clean(path))
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
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
