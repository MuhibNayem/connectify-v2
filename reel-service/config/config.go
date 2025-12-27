package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI string
	DBName   string

	ServerPort string
	GRPCPort   string

	KafkaBrokers []string
	KafkaTopic   string

	UserServiceHost string
	UserServicePort string

	JWTSecret        string
	RedisURLs        []string
	RedisPass        string
	RateLimitEnabled bool
	RateLimitLimit   float64
	RateLimitBurst   int

	CORSAllowedOrigins []string

	JaegerOTLPEndpoint string
}

func Load() *Config {
	godotenv.Load()

	rateLimitEnabled, _ := strconv.ParseBool(getEnv("RATE_LIMIT_ENABLED", "true"))
	rateLimitLimit, _ := strconv.ParseFloat(getEnv("RATE_LIMIT_LIMIT", "50"), 64)
	rateLimitBurst, _ := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "100"))

	corsOrigins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	return &Config{
		MongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:   getEnv("DB_NAME", "messaging_app"),

		ServerPort: getEnv("SERVER_PORT", "8086"),
		GRPCPort:   getEnv("GRPC_PORT", "9096"),

		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "reel-events"),

		UserServiceHost: getEnv("USER_SERVICE_HOST", "localhost"),
		UserServicePort: getEnv("USER_SERVICE_PORT", "9091"),

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

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
