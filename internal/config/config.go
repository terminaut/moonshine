package config

import (
	"os"
	"strings"
)

type Config struct {
	Env          string
	HTTPAddr     string
	JWTKey       string
	PprofEnabled   bool
	TracingEnabled bool
	JaegerEndpoint string
	Database       DatabaseConfig
	Redis          RedisConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Addr     string
	Password string
}

func Load() *Config {
	return &Config{
		Env:          getEnv("ENV", "development"),
		HTTPAddr:     normalizeAddr(getEnv("HTTP_ADDR", ":8080")),
		JWTKey:       getEnv("JWT_KEY", "secret"),
		PprofEnabled:   getEnvBool("PPROF_ENABLED", strings.ToLower(getEnv("ENV", "development")) != "production" && strings.ToLower(getEnv("ENV", "development")) != "prod"),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "localhost:4317"),
		Database: DatabaseConfig{
			Host:     getEnv("DATABASE_HOST", "localhost"),
			Port:     getEnv("DATABASE_PORT", "5433"),
			User:     getEnv("DATABASE_USER", "postgres"),
			Password: getEnv("DATABASE_PASSWORD", "postgres"),
			Name:     getEnv("DATABASE_NAME", "moonshine"),
			SSLMode:  getEnv("DATABASE_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost"),
			Password: getEnv("REDIS_PASSWORD", "secret"),
		},
	}
}

func (c *Config) IsProduction() bool {
	return c.Env == "production" || c.Env == "prod"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func normalizeAddr(addr string) string {
	if addr == "" {
		return addr
	}

	if addr[0] == ':' || addr[0] == '[' {
		return addr
	}

	for _, r := range addr {
		if r < '0' || r > '9' {
			return addr
		}
	}

	return ":" + addr
}
