package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"weatherapi.app/pkg/logger"
)

const (
	GRPCCorrelationIDKey = "correlation-id"
	GRPCRequestIDKey     = "request-id"
)

// GRPCCorrelationInterceptor handles correlation ID for gRPC services
type GRPCCorrelationInterceptor struct {
	logger *logger.Logger
}

// NewGRPCCorrelationInterceptor creates a new gRPC correlation interceptor
func NewGRPCCorrelationInterceptor(logger *logger.Logger) *GRPCCorrelationInterceptor {
	return &GRPCCorrelationInterceptor{
		logger: logger,
	}
}

// UnaryServerInterceptor adds correlation ID tracking to unary gRPC calls
func (i *GRPCCorrelationInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract correlation ID from metadata
		correlationID := i.extractCorrelationID(ctx)

		// Add to context
		ctx = logger.WithCorrelationIDContext(ctx, correlationID)

		// Create request-specific logger
		requestLogger := i.logger.WithCorrelationID(correlationID).WithOperation(info.FullMethod)

		requestLogger.Debug("grpc request started",
			"method", info.FullMethod,
			"correlation_id", correlationID,
		)

		// Call the handler with enhanced context
		resp, err := handler(ctx, req)

		if err != nil {
			requestLogger.Error("grpc request failed",
				"method", info.FullMethod,
				"error", err,
			)
		} else {
			requestLogger.Debug("grpc request completed",
				"method", info.FullMethod,
			)
		}

		return resp, err
	}
}

// UnaryClientInterceptor adds correlation ID to outgoing gRPC calls
func (i *GRPCCorrelationInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Get correlation ID from context
		correlationID := logger.GetCorrelationID(ctx)
		if correlationID == "" {
			ctx, correlationID = logger.GetOrCreateCorrelationID(ctx)
		}

		// Add correlation ID to gRPC metadata
		md := metadata.New(map[string]string{
			GRPCCorrelationIDKey: correlationID,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)

		requestLogger := i.logger.WithCorrelationID(correlationID)

		requestLogger.Debug("grpc client request started",
			"method", method,
			"target", cc.Target(),
		)

		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			requestLogger.Error("grpc client request failed",
				"method", method,
				"target", cc.Target(),
				"error", err,
			)
		} else {
			requestLogger.Debug("grpc client request completed",
				"method", method,
				"target", cc.Target(),
			)
		}

		return err
	}
}

// extractCorrelationID extracts correlation ID from gRPC metadata
func (i *GRPCCorrelationInterceptor) extractCorrelationID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// Try correlation-id first
		if values := md.Get(GRPCCorrelationIDKey); len(values) > 0 {
			return values[0]
		}
		// Try request-id as fallback
		if values := md.Get(GRPCRequestIDKey); len(values) > 0 {
			return values[0]
		}
		// Try x-correlation-id (from HTTP gateway)
		if values := md.Get("x-correlation-id"); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// StreamServerInterceptor adds correlation ID tracking to streaming gRPC calls
func (i *GRPCCorrelationInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()
		correlationID := i.extractCorrelationID(ctx)
		ctx = logger.WithCorrelationIDContext(ctx, correlationID)

		requestLogger := i.logger.WithCorrelationID(correlationID).WithOperation(info.FullMethod)

		requestLogger.Debug("grpc stream started",
			"method", info.FullMethod,
		)

		// Wrap the stream with the enhanced context
		wrappedStream := &correlationServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		err := handler(srv, wrappedStream)

		if err != nil {
			requestLogger.Error("grpc stream failed",
				"method", info.FullMethod,
				"error", err,
			)
		} else {
			requestLogger.Debug("grpc stream completed",
				"method", info.FullMethod,
			)
		}

		return err
	}
}

// correlationServerStream wraps grpc.ServerStream to provide enhanced context
type correlationServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *correlationServerStream) Context() context.Context {
	return s.ctx
}

// LoggerFromGRPCContext extracts correlation-aware logger from gRPC context
func LoggerFromGRPCContext(ctx context.Context, baseLogger *logger.Logger) *logger.Logger {
	if correlationID := logger.GetCorrelationID(ctx); correlationID != "" {
		return baseLogger.WithCorrelationID(correlationID)
	}
	return baseLogger
}

// AddCorrelationToOutgoingGRPC adds correlation ID to outgoing gRPC metadata
func AddCorrelationToOutgoingGRPC(ctx context.Context) context.Context {
	correlationID := logger.GetCorrelationID(ctx)
	if correlationID == "" {
		ctx, correlationID = logger.GetOrCreateCorrelationID(ctx)
	}

	md := metadata.New(map[string]string{
		GRPCCorrelationIDKey: correlationID,
	})

	// Merge with existing metadata if present
	if existing, ok := metadata.FromOutgoingContext(ctx); ok {
		md = metadata.Join(existing, md)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// GRPCServerOptions returns server options with correlation interceptors
func (i *GRPCCorrelationInterceptor) GRPCServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(i.UnaryServerInterceptor()),
		grpc.StreamInterceptor(i.StreamServerInterceptor()),
	}
}

// GRPCDialOptions returns dial options with correlation interceptors
func (i *GRPCCorrelationInterceptor) GRPCDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(i.UnaryClientInterceptor()),
	}
}

// ExtractServiceAndMethod extracts service and method from gRPC full method
func ExtractServiceAndMethod(fullMethod string) (string, string) {
	parts := strings.Split(strings.TrimPrefix(fullMethod, "/"), "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "unknown", "unknown"
}
