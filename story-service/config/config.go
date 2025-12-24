package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// MongoDB
	MongoURI string
	DBName   string

	// gRPC
	GRPCPort string

	// Kafka
	KafkaBrokers []string
	KafkaTopic   string

	// User Service (for author info)
	UserServiceHost string
	UserServicePort string

	// Observability
	JaegerOTLPEndpoint string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		// MongoDB
		MongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:   getEnv("DB_NAME", "messaging_app"),

		// gRPC
		GRPCPort: getEnv("GRPC_PORT", "9097"),

		// Kafka
		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "story-events"),

		// User Service
		UserServiceHost: getEnv("USER_SERVICE_HOST", "localhost"),
		UserServicePort: getEnv("USER_SERVICE_PORT", "9091"),

		JaegerOTLPEndpoint: getEnv("JAEGER_OTLP_ENDPOINT", "localhost:4317"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
