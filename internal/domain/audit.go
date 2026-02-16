package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AuditEntry struct {
	ID         uuid.UUID              `json:"id"`
	Actor      string                 `json:"actor"`
	ActorType  string                 `json:"actor_type"` // management, device, system
	Action     string                 `json:"action"`     // e.g. device.accept, artifact.upload
	Resource   string                 `json:"resource"`   // e.g. device, artifact, deployment
	ResourceID string                 `json:"resource_id"`
	Details    map[string]interface{} `json:"details,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

type AuditFilter struct {
	Actor    *string
	Action   *string
	Resource *string
	Page     int
	PerPage  int
	SortOrder string
}

type AuditRepository interface {
	Create(ctx context.Context, entry *AuditEntry) error
	List(ctx context.Context, filter AuditFilter) ([]*AuditEntry, int, error)
}
