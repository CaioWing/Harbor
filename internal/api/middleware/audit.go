package middleware

import (
	"net/http"
	"strings"

	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/service"
)

// AuditLog returns a middleware that records management API actions.
func AuditLog(auditSvc *service.AuditService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			// Only audit mutating requests that succeeded
			if r.Method == http.MethodGet || r.Method == http.MethodOptions {
				return
			}
			if rw.status >= 400 {
				return
			}

			action, resource := classifyRequest(r.Method, r.URL.Path)
			if action == "" {
				return
			}

			actor := "anonymous"
			if uid, ok := r.Context().Value(UserIDKey).(string); ok && uid != "" {
				actor = uid
			}

			entry := &domain.AuditEntry{
				Actor:     actor,
				ActorType: "management",
				Action:    action,
				Resource:  resource,
				IPAddress: r.RemoteAddr,
				Details:   map[string]interface{}{"method": r.Method, "path": r.URL.Path},
			}

			// Extract resource ID from URL if present
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/management/"), "/")
			if len(parts) >= 2 {
				entry.ResourceID = parts[1]
			}

			auditSvc.Log(r.Context(), entry)
		})
	}
}

func classifyRequest(method, path string) (action, resource string) {
	p := strings.TrimPrefix(path, "/api/v1/management/")

	switch {
	case strings.HasPrefix(p, "devices") && method == http.MethodPut:
		return "device.update_status", "device"
	case strings.HasPrefix(p, "devices") && method == http.MethodPatch:
		return "device.update_tags", "device"
	case strings.HasPrefix(p, "devices") && method == http.MethodDelete:
		return "device.decommission", "device"
	case strings.HasPrefix(p, "artifacts") && method == http.MethodPost:
		return "artifact.upload", "artifact"
	case strings.HasPrefix(p, "artifacts") && method == http.MethodDelete:
		return "artifact.delete", "artifact"
	case strings.HasPrefix(p, "deployments") && method == http.MethodPost && strings.HasSuffix(p, "cancel"):
		return "deployment.cancel", "deployment"
	case strings.HasPrefix(p, "deployments") && method == http.MethodPost:
		return "deployment.create", "deployment"
	case strings.HasPrefix(p, "auth"):
		return "auth.login", "auth"
	default:
		return "", ""
	}
}
