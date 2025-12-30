package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server        ServerConfig
	Storage       StorageConfig
	Queue         QueueConfig
	Channels      ChannelsConfig
	RateLimit     RateLimitConfig
	Retention     RetentionConfig
	Observability ObservabilityConfig
	Redis         RedisConfig
	Auth          AuthConfig
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AuthConfig struct {
	JWTSecret string
	JWTIssuer string
	APIKeys   string // comma-separated key:name pairs
}

type ServerConfig struct {
	HTTPPort       string
	GRPCPort       string
	Environment    string
	UserServiceURL string
}

type StorageConfig struct {
	Type       string // "mongodb", "postgres", "memory"
	URI        string
	Database   string
	Collection string
}

type QueueConfig struct {
	Type    string // "kafka", "rabbitmq", "redis", "memory"
	Brokers []string
	Topic   string
	GroupID string
}

type ChannelsConfig struct {
	InApp   ChannelConfig
	Push    PushConfig
	Email   EmailConfig
	SMS     SMSConfig
	Webhook WebhookConfig
}

type ChannelConfig struct {
	Enabled     bool
	WorkerCount int
	QueueSize   int
}

type PushConfig struct {
	ChannelConfig
	FCMEnabled     bool
	FCMProjectID   string
	FCMCredentials string
	APNSEnabled    bool
	APNSKeyID      string
	APNSTeamID     string
	APNSBundleID   string
	APNSKeyFile    string
	APNSProduction bool
}

type EmailConfig struct {
	ChannelConfig
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	UseTLS       bool
}

type SMSConfig struct {
	ChannelConfig
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
}

type WebhookConfig struct {
	ChannelConfig
	Timeout int
}

type RateLimitConfig struct {
	Enabled       bool
	MaxPerHour    int
	WindowSeconds int
}

type RetentionConfig struct {
	DefaultDays  int
	ReadDays     int
	CleanupHours int
}

type ObservabilityConfig struct {
	MetricsEnabled bool
	MetricsPort    string
	TracingEnabled bool
	JaegerEndpoint string
	LogLevel       string
	LogFormat      string
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			HTTPPort:       getEnv("SERVER_PORT", "8090"),
			GRPCPort:       getEnv("GRPC_PORT", "50060"),
			Environment:    getEnv("ENVIRONMENT", "development"),
			UserServiceURL: getEnv("USER_SERVICE_URL", ""),
		},
		Storage: StorageConfig{
			Type:       getEnv("STORAGE_TYPE", "memory"),
			URI:        getEnv("STORAGE_URI", ""),
			Database:   getEnv("STORAGE_DATABASE", "notifications"),
			Collection: getEnv("STORAGE_COLLECTION", "notifications"),
		},
		Queue: QueueConfig{
			Type:    getEnv("QUEUE_TYPE", "memory"),
			Brokers: []string{getEnv("QUEUE_BROKERS", "localhost:9092")},
			Topic:   getEnv("QUEUE_TOPIC", "notifications"),
			GroupID: getEnv("QUEUE_GROUP_ID", "notification-service"),
		},
		Channels: ChannelsConfig{
			InApp: ChannelConfig{
				Enabled:     getEnvBool("INAPP_ENABLED", true),
				WorkerCount: getEnvInt("INAPP_WORKER_COUNT", 10),
				QueueSize:   getEnvInt("INAPP_QUEUE_SIZE", 10000),
			},
			Push: PushConfig{
				ChannelConfig: ChannelConfig{
					Enabled:     getEnvBool("PUSH_ENABLED", false),
					WorkerCount: getEnvInt("PUSH_WORKER_COUNT", 20),
					QueueSize:   getEnvInt("PUSH_QUEUE_SIZE", 10000),
				},
				FCMEnabled:     getEnvBool("FCM_ENABLED", false),
				FCMProjectID:   getEnv("FCM_PROJECT_ID", ""),
				FCMCredentials: getEnv("FCM_CREDENTIALS", ""),
				APNSEnabled:    getEnvBool("APNS_ENABLED", false),
			},
			Email: EmailConfig{
				ChannelConfig: ChannelConfig{
					Enabled:     getEnvBool("EMAIL_ENABLED", false),
					WorkerCount: getEnvInt("EMAIL_WORKER_COUNT", 15),
					QueueSize:   getEnvInt("EMAIL_QUEUE_SIZE", 5000),
				},
				SMTPHost:     getEnv("SMTP_HOST", ""),
				SMTPPort:     getEnvInt("SMTP_PORT", 587),
				SMTPUsername: getEnv("SMTP_USERNAME", ""),
				SMTPPassword: getEnv("SMTP_PASSWORD", ""),
				FromEmail:    getEnv("SMTP_FROM_EMAIL", ""),
				FromName:     getEnv("SMTP_FROM_NAME", "Notifications"),
				UseTLS:       getEnvBool("SMTP_USE_TLS", true),
			},
			SMS: SMSConfig{
				ChannelConfig: ChannelConfig{
					Enabled:     getEnvBool("SMS_ENABLED", false),
					WorkerCount: getEnvInt("SMS_WORKER_COUNT", 5),
					QueueSize:   getEnvInt("SMS_QUEUE_SIZE", 2000),
				},
				TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
				TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
				TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),
			},
			Webhook: WebhookConfig{
				ChannelConfig: ChannelConfig{
					Enabled:     getEnvBool("WEBHOOK_ENABLED", true),
					WorkerCount: getEnvInt("WEBHOOK_WORKER_COUNT", 10),
					QueueSize:   getEnvInt("WEBHOOK_QUEUE_SIZE", 5000),
				},
				Timeout: getEnvInt("WEBHOOK_TIMEOUT", 30),
			},
		},
		RateLimit: RateLimitConfig{
			Enabled:       getEnvBool("RATE_LIMIT_ENABLED", true),
			MaxPerHour:    getEnvInt("RATE_LIMIT_MAX_PER_HOUR", 100),
			WindowSeconds: getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 3600),
		},
		Retention: RetentionConfig{
			DefaultDays:  getEnvInt("RETENTION_DEFAULT_DAYS", 90),
			ReadDays:     getEnvInt("RETENTION_READ_DAYS", 30),
			CleanupHours: getEnvInt("RETENTION_CLEANUP_HOURS", 1),
		},
		Observability: ObservabilityConfig{
			MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
			MetricsPort:    getEnv("METRICS_PORT", "9102"),
			TracingEnabled: getEnvBool("TRACING_ENABLED", false),
			JaegerEndpoint: getEnv("JAEGER_ENDPOINT", ""),
			LogLevel:       getEnv("LOG_LEVEL", "info"),
			LogFormat:      getEnv("LOG_FORMAT", "json"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", ""),
			JWTIssuer: getEnv("JWT_ISSUER", "notification-service"),
			APIKeys:   getEnv("API_KEYS", ""), // Format: "key1:name1,key2:name2"
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
