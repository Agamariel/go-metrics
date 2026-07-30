package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestTrustedSubnetMiddleware_EmptySubnet(t *testing.T) {
	mw := TrustedSubnetMiddleware("")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestTrustedSubnetMiddleware_InvalidCIDR(t *testing.T) {
	mw := TrustedSubnetMiddleware("not-a-cidr")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestTrustedSubnetMiddleware_AllowedIP_XRealIP(t *testing.T) {
	mw := TrustedSubnetMiddleware("192.168.1.0/24")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.50")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestTrustedSubnetMiddleware_DeniedIP_XRealIP(t *testing.T) {
	mw := TrustedSubnetMiddleware("192.168.1.0/24")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestTrustedSubnetMiddleware_AllowedIP_RemoteAddr(t *testing.T) {
	mw := TrustedSubnetMiddleware("10.0.0.0/8")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.5.6.7:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestTrustedSubnetMiddleware_DeniedIP_RemoteAddr(t *testing.T) {
	mw := TrustedSubnetMiddleware("10.0.0.0/8")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.16.0.1:9999"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestTrustedSubnetMiddleware_InvalidRemoteAddr(t *testing.T) {
	mw := TrustedSubnetMiddleware("10.0.0.0/8")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "bad-addr"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestTrustedSubnetMiddleware_InvalidIPInHeader(t *testing.T) {
	mw := TrustedSubnetMiddleware("10.0.0.0/8")
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "not-an-ip")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
