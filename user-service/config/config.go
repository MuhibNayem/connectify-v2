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
	// Server
	ServerPort       string
	RateLimitEnabled bool
	RateLimitLimit   float64
	RateLimitBurst   int

	// Database
	MongoURI      string
	DBName        string
	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string

	// Redis
	RedisURLs []string
	RedisPass string

	// Kafka
	KafkaBrokers         []string
	UserUpdatedTopic     string
	FriendshipEventTopic string

	// Security
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Observability
	JaegerOTLPEndpoint string

	// Cookies
	RefreshCookieName string
	CookieDomain      string
	CookieSecure      bool
}

func LoadConfig() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Using environment variables directly")
	}

	accessTTL, _ := strconv.Atoi(getEnv("ACCESS_TOKEN_TTL", "5"))      // minutes
	refreshTTL, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_TTL", "1440")) // 24 hours in minutes
	rateLimitEnabled, _ := strconv.ParseBool(getEnv("RATE_LIMIT_ENABLED", "true"))
	rateLimitLimit, _ := strconv.ParseFloat(getEnv("RATE_LIMIT_LIMIT", "50"), 64)
	rateLimitBurst, _ := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "100"))

	cookieSecure, _ := strconv.ParseBool(getEnv("COOKIE_SECURE", "false"))

	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8083"), // Default user-service port
		RateLimitEnabled: rateLimitEnabled,
		RateLimitLimit:   rateLimitLimit,
		RateLimitBurst:   rateLimitBurst,

		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:        getEnv("DB_NAME", "connectify-v2"),
		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "connectify"),

		RedisURLs: strings.Split(getEnv("REDIS_URL", "localhost:6379"), ","),
		RedisPass: getEnv("REDIS_PASS", ""),

		KafkaBrokers:         strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		UserUpdatedTopic:     getEnv("KAFKA_TOPIC_USER_UPDATED", "user-updated"),
		FriendshipEventTopic: getEnv("KAFKA_TOPIC_FRIENDSHIP_EVENTS", "friendship-events"),

		JWTSecret:       getEnv("JWT_SECRET", "very-secret-key"),
		AccessTokenTTL:  time.Minute * time.Duration(accessTTL),
		RefreshTokenTTL: time.Minute * time.Duration(refreshTTL),

		JaegerOTLPEndpoint: getEnv("JAEGER_OTLP_ENDPOINT", "localhost:4317"),

		RefreshCookieName: getEnv("REFRESH_COOKIE_NAME", "connectify_refresh"),
		CookieDomain:      getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:      cookieSecure,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
