package platform

import (
	"context"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/friendship-service/config"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/graph"

	pkgredis "github.com/MuhibNayem/connectify-v2/shared-entity/redis"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitMongo initializes the MongoDB connection
func InitMongo(ctx context.Context, cfg *config.Config) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)
	if cfg.MongoUser != "" && cfg.MongoPassword != "" {
		clientOpts.SetAuth(options.Credential{
			Username: cfg.MongoUser,
			Password: cfg.MongoPassword,
		})
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, err
	}

	slog.Info("Connected to MongoDB", "database", cfg.DBName)
	return client, client.Database(cfg.DBName), nil
}

// InitRedis initializes the Redis connection
func InitRedis(cfg *config.Config) (*pkgredis.ClusterClient, error) {
	client := pkgredis.NewClusterClient(pkgredis.Config{
		RedisURLs: cfg.RedisURLs,
		RedisPass: cfg.RedisPass,
	})
	slog.Info("Connected to Redis", "urls", cfg.RedisURLs)
	return client, nil
}

// InitNeo4j initializes the Neo4j connection
func InitNeo4j(cfg *config.Config) (*graph.Neo4jClient, error) {
	client, err := graph.NewNeo4jClient(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		return nil, err
	}
	slog.Info("Connected to Neo4j", "uri", cfg.Neo4jURI)
	return client, nil
}
