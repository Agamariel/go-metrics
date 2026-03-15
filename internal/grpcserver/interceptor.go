package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TrustedSubnetInterceptor возвращает UnaryServerInterceptor, который проверяет,
// что IP-адрес агента (из метаданных запроса по ключу "x-real-ip") входит в доверенную подсеть.
// При пустом trustedSubnet проверка не выполняется.
func TrustedSubnetInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	if trustedSubnet == "" {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}

	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return nil, status.Errorf(codes.Internal, "некорректная конфигурация trusted_subnet: %v", err)
		}
	}

	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ipStr := extractIP(ctx)
		if ipStr == "" {
			return nil, status.Error(codes.PermissionDenied, "отсутствует заголовок x-real-ip")
		}

		ip := net.ParseIP(ipStr)
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Errorf(codes.PermissionDenied, "IP-адрес %q не входит в доверенную подсеть", ipStr)
		}

		return handler(ctx, req)
	}
}

// extractIP извлекает IP-адрес из метаданных gRPC-запроса по ключу "x-real-ip".
func extractIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("x-real-ip")
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
