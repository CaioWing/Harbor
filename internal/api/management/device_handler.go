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

type DeviceHandler struct {
	deviceSvc *service.DeviceService
}

func NewDeviceHandler(deviceSvc *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceSvc: deviceSvc}
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := response.ParsePagination(r)
	q := r.URL.Query()

	filter := domain.DeviceFilter{
		Page:      page,
		PerPage:   perPage,
		SortBy:    q.Get("sort"),
		SortOrder: q.Get("order"),
	}

	if s := q.Get("status"); s != "" {
		status := domain.DeviceStatus(s)
		filter.Status = &status
	}
	if dt := q.Get("device_type"); dt != "" {
		filter.DeviceType = &dt
	}
	if tags := q["tag"]; len(tags) > 0 {
		filter.Tags = tags
	}

	devices, total, err := h.deviceSvc.List(r.Context(), filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list devices")
		return
	}

	response.Paginated(w, http.StatusOK, devices, page, perPage, total)
}

func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid device id")
		return
	}

	device, err := h.deviceSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "device not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get device")
		return
	}

	response.JSON(w, http.StatusOK, device)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *DeviceHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid device id")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status := domain.DeviceStatus(req.Status)
	switch status {
	case domain.DeviceStatusAccepted, domain.DeviceStatusRejected:
	default:
		response.Error(w, http.StatusBadRequest, "status must be 'accepted' or 'rejected'")
		return
	}

	if err := h.deviceSvc.UpdateStatus(r.Context(), id, status); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "device not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update status")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateTagsRequest struct {
	Tags []string `json:"tags"`
}

func (h *DeviceHandler) UpdateTags(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid device id")
		return
	}

	var req updateTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.deviceSvc.UpdateTags(r.Context(), id, req.Tags); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "device not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update tags")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid device id")
		return
	}

	if err := h.deviceSvc.Decommission(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "device not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to decommission device")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeviceHandler) Count(w http.ResponseWriter, r *http.Request) {
	counts, err := h.deviceSvc.CountByStatus(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to count devices")
		return
	}

	response.JSON(w, http.StatusOK, counts)
}
