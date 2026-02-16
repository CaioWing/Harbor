package management

import (
	"encoding/json"
	"net/http"

	"github.com/CaioWing/Harbor/internal/api/middleware"
	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	jwtMgr       *auth.JWTManager
	adminEmail   string
	adminPassHash string
}

// NewAuthHandler creates a simple auth handler with a single admin user.
// For production, replace with a proper user store.
func NewAuthHandler(jwtMgr *auth.JWTManager) *AuthHandler {
	// Default admin credentials — override via env in production
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	return &AuthHandler{
		jwtMgr:       jwtMgr,
		adminEmail:   "admin@harbor.local",
		adminPassHash: string(hash),
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email != h.adminEmail {
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(h.adminPassHash), []byte(req.Password)); err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, expiresAt, err := h.jwtMgr.Generate("admin")
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response.JSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Refresh generates a new JWT token for an already authenticated user.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		response.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}

	token, expiresAt, err := h.jwtMgr.Generate(userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response.JSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z"),
	})
}
