package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI      string
	MongoUser     string
	MongoPassword string
	DBName        string

	KafkaBrokers []string
	KafkaTopic   string

	RedisURLs []string
	RedisPass string

	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string

	GRPCPort       string
	HTTPPort       string
	PrometheusPort string

	UserServiceURL string

	// Auth
	JWTSecret string

	// CORS
	CORSAllowedOrigins []string

	// Rate Limiting
	RateLimitEnabled bool
	RateLimitLimit   float64
	RateLimitBurst   int

	// Observability
	JaegerOTLPEndpoint string

	// Cache
	CacheTTL time.Duration
}

func LoadConfig() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Using environment variables directly")
	}

	rateLimitEnabled, _ := strconv.ParseBool(getEnv("RATE_LIMIT_ENABLED", "true"))
	rateLimitLimit, _ := strconv.ParseFloat(getEnv("RATE_LIMIT_LIMIT", "100"), 64)
	rateLimitBurst, _ := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "100"))
	cacheTTLMins, _ := strconv.Atoi(getEnv("CACHE_TTL_MINS", "5"))

	return &Config{
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoUser:     getEnv("MONGO_USER", ""),
		MongoPassword: getEnv("MONGO_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "messaging_app"),

		KafkaBrokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "friendship-events"),

		RedisURLs: strings.Split(getEnv("REDIS_URL", "localhost:6379"), ","),
		RedisPass: getEnv("REDIS_PASS", ""),

		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "connectify"),

		GRPCPort:       getEnv("FRIENDSHIP_GRPC_PORT", "9103"),
		HTTPPort:       getEnv("FRIENDSHIP_HTTP_PORT", "8103"),
		PrometheusPort: getEnv("PROMETHEUS_PORT", "9104"),

		UserServiceURL: getEnv("USER_SERVICE_URL", "localhost:9100"),

		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key"),
		CORSAllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),

		RateLimitEnabled: rateLimitEnabled,
		RateLimitLimit:   rateLimitLimit,
		RateLimitBurst:   rateLimitBurst,

		JaegerOTLPEndpoint: getEnv("JAEGER_OTLP_ENDPOINT", "localhost:4317"),

		CacheTTL: time.Duration(cacheTTLMins) * time.Minute,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
