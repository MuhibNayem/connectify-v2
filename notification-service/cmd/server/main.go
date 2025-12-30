package main

import (
	"context"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/config"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/auth"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/channels/email"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/channels/inapp"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/channels/push"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/channels/sms"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/cleanup"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/core"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/dlq"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/idempotency"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/observability"
	kafkaqueue "github.com/MuhibNayem/connectify-v2/notification-service/internal/queue/kafka"
	memoryqueue "github.com/MuhibNayem/connectify-v2/notification-service/internal/queue/memory"
	rabbitmq "github.com/MuhibNayem/connectify-v2/notification-service/internal/queue/rabbitmq"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/ratelimit"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/scheduler"
	grpcserver "github.com/MuhibNayem/connectify-v2/notification-service/internal/server/grpc"
	httpserver "github.com/MuhibNayem/connectify-v2/notification-service/internal/server/http"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/storage/memory"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/storage/mongodb"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/storage/postgres"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/tracing"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func main() {
	log.Println("Starting Universal Notification Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := observability.NewLogger(cfg.Observability.LogLevel, cfg.Observability.LogFormat)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Metrics system initialized via package init
	logger.Info("Metrics system initialized")

	// ==================== STORAGE INITIALIZATION ====================
	var storage adapters.StorageAdapter

	switch cfg.Storage.Type {
	case "mongodb":
		storage, err = mongodb.NewMongoStorage(cfg.Storage.URI, cfg.Storage.Database, cfg.Storage.Collection)
		if err != nil {
			logger.Fatal("Failed to initialize MongoDB", zap.Error(err))
		}
		logger.Info("Using MongoDB storage", zap.String("database", cfg.Storage.Database))

	case "postgres", "postgresql":
		storage, err = postgres.NewPostgresStorage(cfg.Storage.URI, cfg.Storage.MaxOpenConns, cfg.Storage.MaxIdleConns)
		if err != nil {
			logger.Fatal("Failed to initialize PostgreSQL", zap.Error(err))
		}
		logger.Info("Using PostgreSQL storage")

	case "memory":
		storage = memory.NewMemoryStorage()
		logger.Info("Using in-memory storage (development mode)")

	default:
		storage = memory.NewMemoryStorage()
		logger.Warn("Unknown storage type, using in-memory", zap.String("type", cfg.Storage.Type))
	}

	// ==================== SHARED REDIS CLIENT ====================
	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			logger.Warn("Redis not available", zap.Error(err), zap.String("addr", cfg.Redis.Addr))
			redisClient = nil
		} else {
			logger.Info("Redis connected", zap.String("addr", cfg.Redis.Addr))
		}
	}

	// ==================== COMPONENTS INIT ====================

	// Dead Letter Queue Handler
	dlqHandler := dlq.NewHandler(storage, logger)
	logger.Info("DLQ handler initialized")

	// ==================== QUEUE INITIALIZATION ====================
	var queue adapters.QueueAdapter

	switch cfg.Queue.Type {
	case "kafka":
		// Inject DLQ Handler for Poison Pills
		queue = kafkaqueue.NewKafkaQueue(cfg.Queue.Brokers, cfg.Queue.Topic, cfg.Queue.GroupID, func(ctx context.Context, data []byte, err error) error {
			return dlqHandler.SendRaw(ctx, data, err)
		})
		logger.Info("Using Kafka queue (with DLQ enabled)", zap.Strings("brokers", cfg.Queue.Brokers))

	case "rabbitmq":
		if len(cfg.Queue.Brokers) == 0 {
			logger.Fatal("RabbitMQ requires at least one broker URL (amqp://...)")
		}
		// Use first broker as URL
		q, err := rabbitmq.NewRabbitQueue(cfg.Queue.Brokers[0], cfg.Queue.Topic)
		if err != nil {
			logger.Fatal("Failed to init RabbitMQ", zap.Error(err))
		}
		queue = q
		logger.Info("Using RabbitMQ queue", zap.String("url", cfg.Queue.Brokers[0]))

	case "memory":
		queue = memoryqueue.NewMemoryQueue()
		logger.Info("Using in-memory queue (development mode)")

	default:
		queue = memoryqueue.NewMemoryQueue()
		logger.Warn("Unknown queue type, using in-memory", zap.String("type", cfg.Queue.Type))
	}

	// ==================== DLQ QUEUE SETUP (Separate Topic) ====================
	switch cfg.Queue.Type {
	case "kafka":
		dlqTopic := cfg.Queue.Topic + "-dlq"
		dlqQueue := kafkaqueue.NewKafkaQueue(cfg.Queue.Brokers, dlqTopic, cfg.Queue.GroupID, nil)
		dlqHandler.SetQueue(dlqQueue)
		logger.Info("DLQ Queue configured (Kafka)", zap.String("topic", dlqTopic))

	case "rabbitmq":
		if len(cfg.Queue.Brokers) > 0 {
			dlqTopic := cfg.Queue.Topic + "-dlq"
			q, err := rabbitmq.NewRabbitQueue(cfg.Queue.Brokers[0], dlqTopic)
			if err == nil {
				dlqHandler.SetQueue(q)
				logger.Info("DLQ Queue configured (RabbitMQ)", zap.String("topic", dlqTopic))
			} else {
				logger.Error("Failed to init DLQ (RabbitMQ)", zap.Error(err))
			}
		}

	case "memory":
		dlqHandler.SetQueue(memoryqueue.NewMemoryQueue())
	}

	// 3. Distributed Tracing
	tracer := tracing.NewTracer("notification-service")
	logger.Info("Tracing initialized")

	// 5. Delayed Scheduler (For Retries)
	// Pass redisClient (can be nil) and queue
	schedulerService := scheduler.NewService(redisClient, queue, logger)
	logger.Info("Scheduler initialized")

	// 1. Idempotency Service (Exactly-Once Processing)
	var idempotencyService *idempotency.Service
	if redisClient != nil {
		idempotencyService = idempotency.NewService(redisClient, 24*time.Hour)
		logger.Info("Idempotency service initialized (24h TTL)")
	}

	// 4. User Resolver
	var userResolver adapters.UserResolver
	if cfg.Server.UserServiceURL != "" {
		userResolver = adapters.NewHTTPUserResolver(cfg.Server.UserServiceURL)
		logger.Info("Using HTTP User Resolver", zap.String("url", cfg.Server.UserServiceURL))
	} else {
		userResolver = &adapters.DefaultUserResolver{}
		logger.Info("Using Default User Resolver (Notification Data Only)")
	}

	// User Pref Wire-up
	var prefAdapter adapters.UserPreferenceAdapter
	if r, ok := userResolver.(adapters.UserPreferenceAdapter); ok {
		prefAdapter = r
	}

	orchestrator := core.NewOrchestrator(&core.OrchestratorConfig{
		Storage:           storage,
		Queue:             queue,
		UserPrefAdapter:   prefAdapter,
		UserResolver:      userResolver,
		Logger:            logger,
		Idempotency:       idempotencyService,
		DLQHandler:        dlqHandler,
		Tracer:            tracer,
		Scheduler:         schedulerService,
		WorkerConcurrency: cfg.Performance.WorkerConcurrency,
	})

	// ==================== CHANNELS ====================

	// 1. In-App (WebSocket)
	if cfg.Channels.InApp.Enabled {
		wsChannel := inapp.NewWebSocketChannel(inapp.WebSocketConfig{
			RedisClient:         redisClient,
			MaxTotalConnections: 10000, // 10k per instance
			MaxPerUserConns:     5,     // 5 devices per user
		})
		orchestrator.RegisterChannel(wsChannel)
		logger.Info("✅ Channel enabled: In-App (WebSocket)")
	}

	// 2. Email (SMTP)
	if cfg.Channels.Email.Enabled {
		emailCfg := email.EmailConfig{
			Host:     cfg.Channels.Email.SMTPHost,
			Port:     cfg.Channels.Email.SMTPPort,
			Username: cfg.Channels.Email.SMTPUsername,
			Password: cfg.Channels.Email.SMTPPassword,
			From:     cfg.Channels.Email.FromEmail,
			FromName: cfg.Channels.Email.FromName,
			UseTLS:   cfg.Channels.Email.UseTLS,
		}
		emailChannel := email.NewEmailChannel(emailCfg)
		orchestrator.RegisterChannel(emailChannel)
		logger.Info("✅ Channel enabled: Email (SMTP)")
	}

	// 3. SMS (Pluggable)
	if cfg.Channels.SMS.Enabled {
		// Example: Choose provider based on config
		// For now using Twilio as example provider
		provider := sms.NewTwilioProvider(
			cfg.Channels.SMS.TwilioAccountSID,
			cfg.Channels.SMS.TwilioAuthToken,
			cfg.Channels.SMS.TwilioFromNumber,
		)
		smsChannel := sms.NewSMSChannel(provider)
		orchestrator.RegisterChannel(smsChannel)
		logger.Info("✅ Channel enabled: SMS (Twilio Provider)")
	}

	// 4. Push (Pluggable)
	if cfg.Channels.Push.Enabled {
		// Example: Choose provider based on config
		// This shows how easily we can plug in different providers
		var provider push.PushProvider
		if cfg.Channels.Push.FCMEnabled {
			p, err := push.NewFCMProvider(cfg.Channels.Push.FCMProjectID, []byte(cfg.Channels.Push.FCMCredentials))
			if err != nil {
				logger.Error("Failed to init FCM", zap.Error(err))
			} else {
				provider = p
				logger.Info("Channel enabled: Push (FCM Provider)")
			}
		} else if cfg.Channels.Push.APNSEnabled {
			// Read p8 file content
			p8Content, err := os.ReadFile(cfg.Channels.Push.APNSKeyFile)
			if err != nil {
				// Fallback: maybe it's the content itself?
				p8Content = []byte(cfg.Channels.Push.APNSKeyFile)
			}

			p, err := push.NewAPNSProvider(
				cfg.Channels.Push.APNSTeamID,
				cfg.Channels.Push.APNSKeyID,
				cfg.Channels.Push.APNSBundleID,
				string(p8Content),
				cfg.Channels.Push.APNSProduction,
			)
			if err != nil {
				logger.Error("Failed to init APNS", zap.Error(err))
			} else {
				provider = p
				logger.Info("Channel enabled: Push (APNS Provider)")
			}
		}

		if provider != nil {
			pushChannel := push.NewPushChannel(provider)
			orchestrator.RegisterChannel(pushChannel)
		}
	}

	// Start Worker only after channels and resilience primitives are ready
	go func() {
		if err := orchestrator.StartWorker(context.Background()); err != nil {
			logger.Error("Worker failed", zap.Error(err))
		}
	}()

	// ==================== BACKGROUND SERVICES ====================

	// Cleanup Service
	cleanupService := cleanup.NewCleanupService(storage, 1*time.Hour)
	cleanupService.Start()
	logger.Info("Background cleanup service started")

	// ==================== SERVERS ====================

	// Metrics Server
	if cfg.Observability.MetricsEnabled {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.ListenAndServe(":"+cfg.Observability.MetricsPort, nil)
		}()
	}

	// Initialize Auth and Rate Limiter for HTTP server
	var authenticator *auth.Authenticator
	var rateLimiter *ratelimit.Limiter

	// Auth: Only enable if JWT_SECRET is configured
	if cfg.Auth.JWTSecret != "" {
		// Parse API keys from config (format: "key1:name1,key2:name2")
		apiKeys := make(map[string]string)
		if cfg.Auth.APIKeys != "" {
			for _, pair := range strings.Split(cfg.Auth.APIKeys, ",") {
				parts := strings.SplitN(pair, ":", 2)
				if len(parts) == 2 {
					apiKeys[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}

		authenticator = auth.NewAuthenticator(auth.Config{
			JWTSecret: cfg.Auth.JWTSecret,
			JWTIssuer: cfg.Auth.JWTIssuer,
			APIKeys:   apiKeys,
		})
		logger.Info("Authentication initialized", zap.Int("api_keys", len(apiKeys)))
	} else {
		logger.Warn("Authentication disabled (JWT_SECRET not set)")
	}

	// Rate Limiter: Only enable if Redis is available
	if redisClient != nil && cfg.RateLimit.Enabled {
		requestsPerHour := cfg.RateLimit.MaxPerHour
		requestsPerSecond := int(math.Ceil(float64(requestsPerHour) / 3600.0))
		if requestsPerSecond < 1 {
			requestsPerSecond = 1
		}
		burstSize := requestsPerSecond * 2

		rateLimiter = ratelimit.NewLimiter(redisClient, ratelimit.Config{
			RequestsPerSecond: requestsPerSecond,
			BurstSize:         burstSize,
			KeyPrefix:         "ratelimit:notification",
		})
		logger.Info("Rate limiter initialized")
	}

	// HTTP API
	httpServer := httpserver.NewServer(httpserver.ServerConfig{
		Orchestrator:  orchestrator,
		Authenticator: authenticator,
		RateLimiter:   rateLimiter,
	})
	apiAddr := ":" + cfg.Server.HTTPPort

	go func() {
		logger.Info("HTTP server listening", zap.String("addr", apiAddr))
		if err := http.ListenAndServe(apiAddr, httpServer); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// gRPC API
	grpcAddr := ":" + cfg.Server.GRPCPort
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Fatal("Failed to listen for gRPC", zap.Error(err))
	}

	grpcInterceptorMgr := grpcserver.NewInterceptorManager(logger, authenticator)
	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(uint32(cfg.Server.GRPCMaxConcurrentStreams)),
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
		grpc.MaxSendMsgSize(10*1024*1024), // 10MB
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 15 * time.Second,
			MaxConnectionAge:  30 * time.Minute,
			Time:              5 * time.Minute,
			Timeout:           20 * time.Second,
		}),
		grpc.ChainUnaryInterceptor(
			grpcInterceptorMgr.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcInterceptorMgr.StreamServerInterceptor(),
		),
	)
	notificationServer := grpcserver.NewNotificationServer(orchestrator, authenticator != nil)
	notificationServer.Register(grpcServer)

	go func() {
		logger.Info("gRPC server listening", zap.String("addr", grpcAddr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// ==================== SHUTDOWN ====================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")
	cleanupService.Stop()
	grpcServer.GracefulStop()

	logger.Info("Shutdown complete")
}
