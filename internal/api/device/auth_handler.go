package device

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/service"
)

type AuthHandler struct {
	deviceSvc *service.DeviceService
}

func NewAuthHandler(deviceSvc *service.DeviceService) *AuthHandler {
	return &AuthHandler{deviceSvc: deviceSvc}
}

type authRequest struct {
	Identity domain.IdentityData `json:"identity"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Identity) == 0 {
		response.Error(w, http.StatusBadRequest, "identity data is required")
		return
	}

	token, err := h.deviceSvc.Authenticate(r.Context(), req.Identity)
	if err != nil {
		if errors.Is(err, domain.ErrDevicePending) {
			response.Error(w, http.StatusUnauthorized, "device is pending approval")
			return
		}
		if errors.Is(err, domain.ErrDeviceRejected) {
			response.Error(w, http.StatusForbidden, "device has been rejected")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "authentication failed")
		return
	}

	response.JSON(w, http.StatusOK, authResponse{Token: token})
}
