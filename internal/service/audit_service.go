package service

import (
	"context"
	"log/slog"

	"github.com/CaioWing/Harbor/internal/domain"
)

type AuditService struct {
	repo domain.AuditRepository
	log  *slog.Logger
}

func NewAuditService(repo domain.AuditRepository, log *slog.Logger) *AuditService {
	return &AuditService{repo: repo, log: log}
}

// Log records an audit event. It is fire-and-forget: errors are logged but not propagated.
func (s *AuditService) Log(ctx context.Context, entry *domain.AuditEntry) {
	if entry.Details == nil {
		entry.Details = map[string]interface{}{}
	}
	if err := s.repo.Create(ctx, entry); err != nil {
		s.log.Warn("failed to write audit log", "action", entry.Action, "err", err)
	}
}

func (s *AuditService) List(ctx context.Context, filter domain.AuditFilter) ([]*domain.AuditEntry, int, error) {
	return s.repo.List(ctx, filter)
}
