package management

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/service"
)

type DeploymentHandler struct {
	deploySvc *service.DeploymentService
}

func NewDeploymentHandler(deploySvc *service.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{deploySvc: deploySvc}
}

type createDeploymentRequest struct {
	Name              string      `json:"name"`
	ArtifactID        string      `json:"artifact_id"`
	TargetDeviceIDs   []string    `json:"target_device_ids,omitempty"`
	TargetDeviceTags  []string    `json:"target_device_tags,omitempty"`
	TargetDeviceTypes []string    `json:"target_device_types,omitempty"`
	MaxParallel       int         `json:"max_parallel"`
}

func (h *DeploymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	artifactID, err := uuid.Parse(req.ArtifactID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid artifact_id")
		return
	}

	var deviceIDs []uuid.UUID
	for _, idStr := range req.TargetDeviceIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid device id: "+idStr)
			return
		}
		deviceIDs = append(deviceIDs, id)
	}

	input := service.CreateDeploymentInput{
		Name:              req.Name,
		ArtifactID:        artifactID,
		TargetDeviceIDs:   deviceIDs,
		TargetDeviceTags:  req.TargetDeviceTags,
		TargetDeviceTypes: req.TargetDeviceTypes,
		MaxParallel:       req.MaxParallel,
	}

	deployment, err := h.deploySvc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "artifact not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create deployment")
		return
	}

	response.JSON(w, http.StatusCreated, deployment)
}

func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := response.ParsePagination(r)
	q := r.URL.Query()

	filter := domain.DeploymentFilter{
		Page:      page,
		PerPage:   perPage,
		SortBy:    q.Get("sort"),
		SortOrder: q.Get("order"),
	}

	if s := q.Get("status"); s != "" {
		status := domain.DeploymentStatus(s)
		filter.Status = &status
	}

	deployments, total, err := h.deploySvc.List(r.Context(), filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list deployments")
		return
	}

	response.Paginated(w, http.StatusOK, deployments, page, perPage, total)
}

func (h *DeploymentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deployment id")
		return
	}

	deployment, err := h.deploySvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "deployment not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get deployment")
		return
	}

	response.JSON(w, http.StatusOK, deployment)
}

func (h *DeploymentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deployment id")
		return
	}

	if err := h.deploySvc.Cancel(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "deployment not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to cancel deployment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeploymentHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid deployment id")
		return
	}

	devices, err := h.deploySvc.GetDeploymentDevices(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get deployment devices")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{"data": devices})
}

func (h *DeploymentHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.deploySvc.GetStats(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	response.JSON(w, http.StatusOK, stats)
}
