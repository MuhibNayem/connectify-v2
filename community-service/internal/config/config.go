package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI    string
	DBName      string
	GRPCPort    string
	MetricsPort string
}

func LoadConfig() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Using environment variables directly")
	}

	return &Config{
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:      getEnv("DB_NAME", "messaging_app"),
		GRPCPort:    getEnv("COMMUNITY_GRPC_PORT", "9101"),
		MetricsPort: getEnv("COMMUNITY_METRICS_PORT", "9102"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
