package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI       string
	CassandraHosts []string
	GRPCPort       string
	MetricsPort    string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	grpcPort := getEnv("GRPC_PORT", "9097")
	metricsPort := getEnv("METRICS_PORT", "9198")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017/messaging_app")

	return &Config{
		MongoURI:       mongoURI,
		CassandraHosts: []string{getEnv("CASSANDRA_HOSTS", "localhost:9042")},
		GRPCPort:       grpcPort,
		MetricsPort:    metricsPort,
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
