package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI       string
	DBName         string
	Neo4jURI       string
	Neo4jUser      string
	Neo4jPassword  string
	GRPCPort       string
	UserServiceURL string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Using environment variables directly (no .env file found)")
	}

	return &Config{
		MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:         getEnv("DB_NAME", "messaging_app"),
		Neo4jURI:       getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:      getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword:  getEnv("NEO4J_PASSWORD", "connectify"),
		GRPCPort:       getEnv("FRIENDSHIP_GRPC_PORT", "9103"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "localhost:9100"), // Default port for user-service
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
