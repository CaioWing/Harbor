package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/CaioWing/Harbor/internal/service"
)

func DeviceAuth(deviceSvc *service.DeviceService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			if token == header {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			device, err := deviceSvc.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, `{"error":"invalid device token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), DeviceIDKey, device.ID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
