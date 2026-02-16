package management

import (
	"net/http"

	"github.com/CaioWing/Harbor/internal/api/response"
	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/service"
)

type AuditHandler struct {
	auditSvc *service.AuditService
}

func NewAuditHandler(auditSvc *service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := response.ParsePagination(r)

	filter := domain.AuditFilter{
		Page:    page,
		PerPage: perPage,
	}

	if v := r.URL.Query().Get("actor"); v != "" {
		filter.Actor = &v
	}
	if v := r.URL.Query().Get("action"); v != "" {
		filter.Action = &v
	}
	if v := r.URL.Query().Get("resource"); v != "" {
		filter.Resource = &v
	}
	if v := r.URL.Query().Get("order"); v != "" {
		filter.SortOrder = v
	}

	entries, total, err := h.auditSvc.List(r.Context(), filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list audit log")
		return
	}

	response.Paginated(w, http.StatusOK, entries, page, perPage, total)
}
