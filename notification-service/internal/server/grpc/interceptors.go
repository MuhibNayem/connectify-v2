package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InterceptorManager holds dependencies for interceptors
type InterceptorManager struct {
	logger        *zap.Logger
	authenticator *auth.Authenticator
}

func NewInterceptorManager(logger *zap.Logger, authenticator *auth.Authenticator) *InterceptorManager {
	return &InterceptorManager{
		logger:        logger,
		authenticator: authenticator,
	}
}

// UnaryServerInterceptor returns a new unary server interceptor with logging and recovery
func (im *InterceptorManager) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// 1. Panic Recovery
		defer func() {
			if r := recover(); r != nil {
				im.logger.Error("Panic in gRPC handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			}
		}()

		if im.authenticator != nil {
			var authErr error
			ctx, authErr = im.authenticator.AuthenticateGRPC(ctx)
			if authErr != nil {
				im.logger.Warn("gRPC authentication failed",
					zap.String("method", info.FullMethod),
					zap.Error(authErr),
				)
				return nil, status.Error(codes.Unauthenticated, authErr.Error())
			}
		}

		resp, err := handler(ctx, req)

		// 3. Logging
		duration := time.Since(start)
		code := status.Code(err)

		im.logger.Info("gRPC Request",
			zap.String("method", info.FullMethod),
			zap.String("status", code.String()),
			zap.Duration("duration", duration),
			zap.Error(err),
		)

		return resp, err
	}
}

// StreamServerInterceptor returns a new stream server interceptor with logging and recovery
func (im *InterceptorManager) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		defer func() {
			if r := recover(); r != nil {
				im.logger.Error("Panic in gRPC stream handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())),
				)
			}
		}()

		if im.authenticator != nil {
			authCtx, authErr := im.authenticator.AuthenticateGRPC(ss.Context())
			if authErr != nil {
				im.logger.Warn("gRPC stream authentication failed",
					zap.String("method", info.FullMethod),
					zap.Error(authErr),
				)
				return status.Error(codes.Unauthenticated, authErr.Error())
			}
			ss = &authenticatedServerStream{
				ServerStream: ss,
				ctx:          authCtx,
			}
		}

		err := handler(srv, ss)

		duration := time.Since(start)
		code := status.Code(err)

		im.logger.Info("gRPC Stream",
			zap.String("method", info.FullMethod),
			zap.String("status", code.String()),
			zap.Duration("duration", duration),
			zap.Error(err),
		)

		return err
	}
}

type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}
