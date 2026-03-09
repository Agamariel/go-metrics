package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет, что IP клиента (из заголовка X-Real-IP или из r.RemoteAddr) входит в доверенную подсеть (CIDR).
// При пустом значении trustedSubnet проверка не выполняется.
func TrustedSubnetMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if trustedSubnet == "" {
			return next
		}

		_, subnet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "server misconfigured: invalid trusted_subnet", http.StatusInternalServerError)
			})
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipStr := r.Header.Get("X-Real-IP")
			if ipStr == "" {
				// Извлекаем IP из r.RemoteAddr (формат "ip:port")
				host, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
				ipStr = host
			}

			ip := net.ParseIP(ipStr)
			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
