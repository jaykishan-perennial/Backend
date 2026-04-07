package config

import "os"

type Config struct {
	Port      string
	JWTSecret string
	DBPath    string
}

func Load() *Config {
	return &Config{
		Port:      getEnv("PORT", "8080"),
		JWTSecret: getEnv("JWT_SECRET", "license-management-secret-key"),
		DBPath:    getEnv("DB_PATH", "license_management.db"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
