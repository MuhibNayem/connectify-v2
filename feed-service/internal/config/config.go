package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort        string
	GRPCPort          string
	MongoURI          string
	DBName            string
	RedisAddrs        []string
	KafkaBrokers      []string
	KafkaTopic        string
	NotificationTopic string
	WebSocketTopic    string
	Neo4jURI          string
	Neo4jUser         string
	Neo4jPassword     string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "8082"), // Different from messaging-app
		GRPCPort:          getEnv("GRPC_PORT", "9098"),   // New port for feed-service
		MongoURI:          getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:            getEnv("DB_NAME", "messaging_app"), // Shared DB
		RedisAddrs:        []string{getEnv("REDIS_ADDR", "localhost:6379")},
		KafkaBrokers:      []string{getEnv("KAFKA_BROKER", "localhost:9092")},
		KafkaTopic:        getEnv("KAFKA_TOPIC", "messages"),
		NotificationTopic: getEnv("NOTIFICATION_TOPIC", "notifications_events"),
		// Neo4j Config
		Neo4jURI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "connectify"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
