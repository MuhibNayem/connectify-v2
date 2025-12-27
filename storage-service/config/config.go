package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPPort         string
	GRPCPort         string
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool
	StoragePublicURL string
	ArchiveBucket    string
	PrometheusPort   string
}

func LoadConfig() *Config {
	return &Config{
		HTTPPort:         getEnv("HTTP_PORT", "8087"),
		GRPCPort:         getEnv("GRPC_PORT", "9087"),
		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", "minio:9000"),
		StorageAccessKey: getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey: getEnv("STORAGE_SECRET_KEY", "minioadmin"),
		StorageBucket:    getEnv("STORAGE_BUCKET", "connectify-uploads"),
		StorageUseSSL:    getEnvBool("STORAGE_USE_SSL", false),
		StoragePublicURL: getEnv("STORAGE_PUBLIC_URL", "http://localhost:9000"),
		ArchiveBucket:    getEnv("ARCHIVE_BUCKET", "connectify-archive"),
		PrometheusPort:   getEnv("PROMETHEUS_PORT", "9187"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, _ := strconv.ParseBool(value)
		return b
	}
	return defaultValue
}
