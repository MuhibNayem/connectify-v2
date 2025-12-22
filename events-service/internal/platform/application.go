package platform

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gitlab.com/spydotech-group/events-service/config"
	"gitlab.com/spydotech-group/events-service/internal/cache"
	"gitlab.com/spydotech-group/events-service/internal/controllers"
	"gitlab.com/spydotech-group/events-service/internal/graph"
	"gitlab.com/spydotech-group/events-service/internal/integration"
	"gitlab.com/spydotech-group/events-service/internal/producer"
	"gitlab.com/spydotech-group/events-service/internal/repository"
	"gitlab.com/spydotech-group/events-service/internal/service"

	pkgkafka "gitlab.com/spydotech-group/shared-entity/kafka"
	"gitlab.com/spydotech-group/shared-entity/redis"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type Application struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg     *config.Config
	metrics *config.Metrics

	mongoClient *mongo.Client
	db          *mongo.Database
	redisClient *redis.ClusterClient
	neo4jClient *graph.Neo4jClient

	dlqProducer   *pkgkafka.DLQProducer
	eventProducer *producer.EventProducer

	eventService  *service.EventService
	mainRouter    *gin.Engine
	httpServer    *http.Server
	metricsServer *http.Server

	shutdownOnce sync.Once
}

func NewApplication(parentCtx context.Context, cfg *config.Config, metrics *config.Metrics) (*Application, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	app := &Application{
		ctx:     ctx,
		cancel:  cancel,
		cfg:     cfg,
		metrics: metrics,
	}

	if err := app.bootstrap(); err != nil {
		app.Close()
		return nil, err
	}
	return app, nil
}

func (a *Application) Run() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 4)
	startServer := func(srv *http.Server, name string) {
		go func() {
			log.Printf("%s starting on %s", name, srv.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("%s failed: %w", name, err)
			}
		}()
	}

	if a.httpServer != nil {
		startServer(a.httpServer, "HTTP server")
	}
	startServer(a.metricsServer, "Metrics server")

	select {
	case <-quit:
		log.Println("Received shutdown signal")
		return a.Shutdown()
	case err := <-errCh:
		log.Printf("Server error: %v", err)
		return a.Shutdown()
	}
}

func (a *Application) Shutdown() error {
	var shutdownErr error
	a.shutdownOnce.Do(func() {
		log.Println("Shutting down application...")
		a.cancel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if a.httpServer != nil {
			if err := a.httpServer.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown error: %v", err)
				shutdownErr = err
			}
		}
		if err := a.metricsServer.Shutdown(ctx); err != nil {
			log.Printf("Metrics server shutdown error: %v", err)
			shutdownErr = err
		}

		a.Close()
	})
	return shutdownErr
}

func (a *Application) Close() {
	if a.dlqProducer != nil {
		a.dlqProducer.Close()
	}
	if a.eventProducer != nil {
		a.eventProducer.Close()
	}
	if a.neo4jClient != nil {
		_ = a.neo4jClient.Close(context.Background())
	}
	if a.redisClient != nil {
		_ = a.redisClient.Close()
	}
	if a.mongoClient != nil {
		_ = a.mongoClient.Disconnect(context.Background())
	}
}

func (a *Application) bootstrap() error {
	var err error
	a.mongoClient, a.db, err = InitMongo(a.ctx, a.cfg)
	if err != nil {
		return err
	}

	a.redisClient, err = InitRedis(a.cfg)
	if err != nil {
		return err
	}

	a.neo4jClient, err = InitNeo4j(a.cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to Neo4j: %v", err)
	}

	a.dlqProducer = pkgkafka.NewDLQProducer(a.cfg.KafkaBrokers)

	eventRepo := repository.NewEventRepository(a.db)
	userLocalRepo := integration.NewUserLocalRepository(a.db)

	// Graph & other repos
	eventGraphRepo := repository.NewEventGraphRepository(a.neo4jClient.Driver)
	eventInvitationRepo := repository.NewEventInvitationRepository(a.db)
	eventPostRepo := repository.NewEventPostRepository(a.db)
	friendshipRepo := integration.NewFriendshipLocalRepository(a.db)

	notificationProducer := producer.NewNotificationProducer(a.cfg.KafkaBrokers, "notifications")
	a.eventProducer = producer.NewEventProducer(a.cfg.KafkaBrokers, a.cfg.KafkaTopic)
	eventCache := cache.NewEventCache(a.redisClient)

	a.eventService = service.NewEventService(
		eventRepo,
		userLocalRepo,
		eventGraphRepo,
		eventInvitationRepo,
		eventPostRepo,
		notificationProducer,
		eventCache,
		a.eventProducer,
	)

	eventRecommendationService := service.NewEventRecommendationService(
		eventRepo,
		eventGraphRepo,
		userLocalRepo,
		friendshipRepo,
		eventCache,
	)

	// Controller initialization
	eventController := controllers.NewEventController(a.eventService, eventRecommendationService)
	routerConfig := RouterConfig{
		EventController: eventController,
	}
	a.mainRouter = a.buildRouters(routerConfig)

	a.httpServer = &http.Server{
		Addr:    net.JoinHostPort("", a.cfg.ServerPort),
		Handler: a.mainRouter,
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", config.MetricsHandler())
	a.metricsServer = &http.Server{
		Addr:    net.JoinHostPort("", a.cfg.PrometheusPort),
		Handler: metricsMux,
	}
	return nil
}
