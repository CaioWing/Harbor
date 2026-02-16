package device

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/api/middleware"
	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/service"
)

type InventoryHandler struct {
	deviceSvc *service.DeviceService
}

func NewInventoryHandler(deviceSvc *service.DeviceService) *InventoryHandler {
	return &InventoryHandler{deviceSvc: deviceSvc}
}

func (h *InventoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	deviceIDStr, _ := r.Context().Value(middleware.DeviceIDKey).(string)
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid device context")
		return
	}

	var inventory map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&inventory); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.deviceSvc.UpdateInventory(r.Context(), deviceID, inventory); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update inventory")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
