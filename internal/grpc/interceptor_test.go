package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// passHandler — простой обработчик, который возвращает nil без ошибок.
func passHandler(ctx context.Context, req any) (any, error) {
	return "ok", nil
}

func callInterceptor(interceptor grpc.UnaryServerInterceptor, ctx context.Context) error {
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, passHandler)
	return err
}

func TestTrustedSubnetInterceptor_EmptySubnet(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("")
	// При пустой подсети проверка не выполняется — любой IP проходит
	ctx := context.Background()
	err := callInterceptor(interceptor, ctx)
	assert.NoError(t, err)
}

func TestTrustedSubnetInterceptor_InvalidCIDR(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("not-a-cidr")
	ctx := context.Background()
	err := callInterceptor(interceptor, ctx)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestTrustedSubnetInterceptor_NoIPHeader(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")
	// Контекст без метаданных
	ctx := context.Background()
	err := callInterceptor(interceptor, ctx)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestTrustedSubnetInterceptor_EmptyIPInMetadata(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")
	// Метаданные есть, но ключа x-real-ip нет
	md := metadata.New(map[string]string{"other-key": "value"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	err := callInterceptor(interceptor, ctx)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestTrustedSubnetInterceptor_IPNotInSubnet(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")
	md := metadata.New(map[string]string{"x-real-ip": "10.0.0.1"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	err := callInterceptor(interceptor, ctx)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestTrustedSubnetInterceptor_IPInSubnet(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")
	md := metadata.New(map[string]string{"x-real-ip": "192.168.1.42"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	err := callInterceptor(interceptor, ctx)
	assert.NoError(t, err)
}

func TestTrustedSubnetInterceptor_InvalidIPFormat(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")
	md := metadata.New(map[string]string{"x-real-ip": "not-an-ip"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	err := callInterceptor(interceptor, ctx)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestExtractIP_NoMetadata(t *testing.T) {
	ip := extractIP(context.Background())
	assert.Equal(t, "", ip)
}

func TestExtractIP_WithMetadata(t *testing.T) {
	md := metadata.New(map[string]string{"x-real-ip": "10.0.0.5"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ip := extractIP(ctx)
	assert.Equal(t, "10.0.0.5", ip)
}

func TestExtractIP_MissingKey(t *testing.T) {
	md := metadata.New(map[string]string{"other": "value"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ip := extractIP(ctx)
	assert.Equal(t, "", ip)
}
