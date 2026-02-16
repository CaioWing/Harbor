package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/api/middleware"
	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/service"
)

type DeploymentHandler struct {
	deploySvc   *service.DeploymentService
	artifactSvc *service.ArtifactService
}

func NewDeploymentHandler(deploySvc *service.DeploymentService, artifactSvc *service.ArtifactService) *DeploymentHandler {
	return &DeploymentHandler{deploySvc: deploySvc, artifactSvc: artifactSvc}
}

type nextDeploymentResponse struct {
	DeploymentID string           `json:"deployment_id"`
	DDID         string           `json:"dd_id"`
	Artifact     artifactResponse `json:"artifact"`
	Retry        retryConfig      `json:"retry"`
}

type retryConfig struct {
	MaxAttempts int `json:"max_attempts"`
	IntervalSec int `json:"interval_sec"`
	BackoffMul  int `json:"backoff_multiplier"`
}

type artifactResponse struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	TargetPath     string `json:"target_path"`
	FileMode       string `json:"file_mode"`
	ChecksumSHA256 string `json:"checksum_sha256"`
	FileSize       int64  `json:"file_size"`
	DownloadURL    string `json:"download_url"`
	PreInstallCmd  string `json:"pre_install_cmd,omitempty"`
	PostInstallCmd string `json:"post_install_cmd,omitempty"`
}

func (h *DeploymentHandler) GetNext(w http.ResponseWriter, r *http.Request) {
	deviceIDStr, _ := r.Context().Value(middleware.DeviceIDKey).(string)
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid device context")
		return
	}

	dd, dep, art, err := h.deploySvc.GetNextForDevice(r.Context(), deviceID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to check deployments")
		return
	}

	response.JSON(w, http.StatusOK, nextDeploymentResponse{
		DeploymentID: dep.ID.String(),
		DDID:         dd.ID.String(),
		Artifact: artifactResponse{
			Name:           art.Name,
			Version:        art.Version,
			TargetPath:     art.TargetPath,
			FileMode:       art.FileMode,
			ChecksumSHA256: art.ChecksumSHA256,
			FileSize:       art.FileSize,
			DownloadURL:    fmt.Sprintf("/api/v1/device/deployments/%s/download", dd.ID),
			PreInstallCmd:  art.PreInstallCmd,
			PostInstallCmd: art.PostInstallCmd,
		},
		Retry: retryConfig{
			MaxAttempts: 3,
			IntervalSec: 30,
			BackoffMul:  2, // exponential backoff: 30s, 60s, 120s
		},
	})
}

type statusUpdateRequest struct {
	Status string `json:"status"`
	Log    string `json:"log"`
}

func (h *DeploymentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ddIDStr := chi.URLParam(r, "id")
	ddID, err := uuid.Parse(ddIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deployment device id")
		return
	}

	var req statusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status := domain.DeploymentDeviceStatus(req.Status)
	switch status {
	case domain.DDStatusDownloading, domain.DDStatusInstalling,
		domain.DDStatusSuccess, domain.DDStatusFailure:
	default:
		response.Error(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err := h.deploySvc.UpdateDeviceStatus(r.Context(), ddID, status, req.Log); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "deployment device not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update status")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeploymentHandler) Download(w http.ResponseWriter, r *http.Request) {
	ddIDStr := chi.URLParam(r, "id")
	_, err := uuid.Parse(ddIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	// For now, extract artifact from the deployment context
	// In a full implementation, we'd validate the dd belongs to the requesting device
	deviceIDStr, _ := r.Context().Value(middleware.DeviceIDKey).(string)
	deviceID, _ := uuid.Parse(deviceIDStr)

	dd, _, art, err := h.deploySvc.GetNextForDevice(r.Context(), deviceID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "no pending deployment")
		return
	}
	_ = dd

	reader, artifact, err := h.artifactSvc.OpenFile(r.Context(), art.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to open artifact")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, artifact.FileName))
	w.Header().Set("X-Checksum-SHA256", artifact.ChecksumSHA256)
	http.ServeContent(w, r, artifact.FileName, artifact.CreatedAt, nil)

	// Stream the file
	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
}
