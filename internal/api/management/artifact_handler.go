package management

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/service"
)

type ArtifactHandler struct {
	artifactSvc *service.ArtifactService
}

func NewArtifactHandler(artifactSvc *service.ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{artifactSvc: artifactSvc}
}

func (h *ArtifactHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := response.ParsePagination(r)
	q := r.URL.Query()

	filter := domain.ArtifactFilter{
		Page:      page,
		PerPage:   perPage,
		SortBy:    q.Get("sort"),
		SortOrder: q.Get("order"),
	}

	if name := q.Get("name"); name != "" {
		filter.Name = &name
	}
	if dt := q.Get("device_type"); dt != "" {
		filter.DeviceType = &dt
	}

	artifacts, total, err := h.artifactSvc.List(r.Context(), filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list artifacts")
		return
	}

	response.Paginated(w, http.StatusOK, artifacts, page, perPage, total)
}

func (h *ArtifactHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid artifact id")
		return
	}

	artifact, err := h.artifactSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "artifact not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get artifact")
		return
	}

	response.JSON(w, http.StatusOK, artifact)
}

func (h *ArtifactHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Max 500MB
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	deviceTypes := strings.Split(r.FormValue("device_types"), ",")
	for i := range deviceTypes {
		deviceTypes[i] = strings.TrimSpace(deviceTypes[i])
	}

	input := service.CreateArtifactInput{
		Name:           r.FormValue("name"),
		Version:        r.FormValue("version"),
		Description:    r.FormValue("description"),
		FileName:       header.Filename,
		TargetPath:     r.FormValue("target_path"),
		FileMode:       r.FormValue("file_mode"),
		FileOwner:      r.FormValue("file_owner"),
		DeviceTypes:    deviceTypes,
		PreInstallCmd:  r.FormValue("pre_install_cmd"),
		PostInstallCmd: r.FormValue("post_install_cmd"),
		RollbackCmd:    r.FormValue("rollback_cmd"),
		File:           file,
	}

	artifact, err := h.artifactSvc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			response.Error(w, http.StatusConflict, "artifact with this name and version already exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to create artifact")
		return
	}

	response.JSON(w, http.StatusCreated, artifact)
}

func (h *ArtifactHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid artifact id")
		return
	}

	reader, artifact, err := h.artifactSvc.OpenFile(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "artifact not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to open artifact")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, artifact.FileName))
	w.Header().Set("X-Checksum-SHA256", artifact.ChecksumSHA256)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", artifact.FileSize))

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

func (h *ArtifactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid artifact id")
		return
	}

	if err := h.artifactSvc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "artifact not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete artifact")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
