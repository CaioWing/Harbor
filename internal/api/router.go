package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/CaioWing/Harbor/internal/api/device"
	"github.com/CaioWing/Harbor/internal/api/management"
	"github.com/CaioWing/Harbor/internal/api/middleware"
	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/auth"
	"github.com/CaioWing/Harbor/internal/service"
)

type RouterDeps struct {
	DeviceSvc     *service.DeviceService
	ArtifactSvc   *service.ArtifactService
	DeploymentSvc *service.DeploymentService
	AuditSvc      *service.AuditService
	JWTManager    *auth.JWTManager
	CORSOrigins   string
	Logger        *slog.Logger
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Metrics
	metrics := middleware.NewMetrics()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.Logger(deps.Logger))
	r.Use(metrics.Middleware())

	// CORS
	origins := strings.Split(deps.CORSOrigins, ",")
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"X-Checksum-SHA256", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Prometheus metrics
	r.Get("/metrics", metrics.Handler())

	// Device API — used by harbor-agent on devices
	deviceAuthHandler := device.NewAuthHandler(deps.DeviceSvc)
	deviceDeployHandler := device.NewDeploymentHandler(deps.DeploymentSvc, deps.ArtifactSvc)
	deviceInventoryHandler := device.NewInventoryHandler(deps.DeviceSvc)

	r.Route("/api/v1/device", func(r chi.Router) {
		// Rate limit device polling: 10 req/s with burst of 20
		r.Use(middleware.RateLimit(10, 20))

		// Auth endpoint (no token required)
		r.Post("/auth", deviceAuthHandler.Authenticate)

		// Authenticated device endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.DeviceAuth(deps.DeviceSvc))
			r.Get("/deployments/next", deviceDeployHandler.GetNext)
			r.Put("/deployments/{id}/status", deviceDeployHandler.UpdateStatus)
			r.Get("/deployments/{id}/download", deviceDeployHandler.Download)
			r.Patch("/inventory", deviceInventoryHandler.Update)
		})
	})

	// Management API — used by React frontend
	mgmtAuthHandler := management.NewAuthHandler(deps.JWTManager)
	mgmtDeviceHandler := management.NewDeviceHandler(deps.DeviceSvc)
	mgmtArtifactHandler := management.NewArtifactHandler(deps.ArtifactSvc)
	mgmtDeploymentHandler := management.NewDeploymentHandler(deps.DeploymentSvc)
	mgmtAuditHandler := management.NewAuditHandler(deps.AuditSvc)

	r.Route("/api/v1/management", func(r chi.Router) {
		// Rate limit management API: 30 req/s with burst of 60
		r.Use(middleware.RateLimit(30, 60))

		// Login (no auth required)
		r.Post("/auth/login", mgmtAuthHandler.Login)

		// Refresh token (requires valid JWT)
		r.Group(func(r chi.Router) {
			r.Use(middleware.ManagementAuth(deps.JWTManager))
			r.Post("/auth/refresh", mgmtAuthHandler.Refresh)
		})

		// Authenticated management endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.ManagementAuth(deps.JWTManager))
			r.Use(middleware.AuditLog(deps.AuditSvc))

			// Devices
			r.Get("/devices", mgmtDeviceHandler.List)
			r.Get("/devices/count", mgmtDeviceHandler.Count)
			r.Get("/devices/{id}", mgmtDeviceHandler.Get)
			r.Put("/devices/{id}/status", mgmtDeviceHandler.UpdateStatus)
			r.Patch("/devices/{id}/tags", mgmtDeviceHandler.UpdateTags)
			r.Delete("/devices/{id}", mgmtDeviceHandler.Delete)

			// Artifacts
			r.Get("/artifacts", mgmtArtifactHandler.List)
			r.Post("/artifacts", mgmtArtifactHandler.Upload)
			r.Get("/artifacts/{id}", mgmtArtifactHandler.Get)
			r.Get("/artifacts/{id}/download", mgmtArtifactHandler.Download)
			r.Delete("/artifacts/{id}", mgmtArtifactHandler.Delete)

			// Deployments
			r.Get("/deployments", mgmtDeploymentHandler.List)
			r.Post("/deployments", mgmtDeploymentHandler.Create)
			r.Get("/deployments/statistics", mgmtDeploymentHandler.Stats)
			r.Get("/deployments/{id}", mgmtDeploymentHandler.Get)
			r.Post("/deployments/{id}/cancel", mgmtDeploymentHandler.Cancel)
			r.Get("/deployments/{id}/devices", mgmtDeploymentHandler.GetDevices)

			// Audit Log
			r.Get("/audit", mgmtAuditHandler.List)
		})
	})

	return r
}
