package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI       string
	CassandraHosts []string
	KafkaBrokers   []string

	GRPCPort    string
	ServerPort  string
	MetricsPort string

	JWTSecret string

	RedisURLs          []string
	RedisPass          string
	RateLimitEnabled   bool
	RateLimitLimit     float64
	RateLimitBurst     int
	CORSAllowedOrigins []string

	// Observability
	JaegerOTLPEndpoint string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	grpcPort := getEnv("GRPC_PORT", "9097")
	serverPort := getEnv("SERVER_PORT", "8087")
	metricsPort := getEnv("METRICS_PORT", "9198")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017/messaging_app")

	rateLimitEnabled, _ := strconv.ParseBool(getEnv("RATE_LIMIT_ENABLED", "true"))
	rateLimitLimit, _ := strconv.ParseFloat(getEnv("RATE_LIMIT_LIMIT", "50"), 64)
	rateLimitBurst, _ := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "100"))

	corsOrigins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	return &Config{
		MongoURI:           mongoURI,
		CassandraHosts:     []string{getEnv("CASSANDRA_HOSTS", "localhost:9042")},
		KafkaBrokers:       strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		GRPCPort:           grpcPort,
		ServerPort:         serverPort,
		MetricsPort:        metricsPort,
		JWTSecret:          getEnv("JWT_SECRET", "very-secret-key"),
		RedisURLs:          strings.Split(getEnv("REDIS_URL", "localhost:6379"), ","),
		RedisPass:          getEnv("REDIS_PASS", ""),
		RateLimitEnabled:   rateLimitEnabled,
		RateLimitLimit:     rateLimitLimit,
		RateLimitBurst:     rateLimitBurst,
		CORSAllowedOrigins: corsOrigins,
		JaegerOTLPEndpoint: getEnv("JAEGER_OTLP_ENDPOINT", "localhost:4317"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
