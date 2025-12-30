package observability

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a production-ready structured logger
func NewLogger(level, format string) (*zap.Logger, error) {
	var config zap.Config

	// Choose config based on format
	if format == "json" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Add caller information
	config.EncoderConfig.CallerKey = "caller"
	config.DisableStacktrace = false

	return config.Build()
}

// NewSugaredLogger creates a sugared logger for easier usage
func NewSugaredLogger(level, format string) (*zap.SugaredLogger, error) {
	logger, err := NewLogger(level, format)
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}
