package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CaioWing/Harbor/internal/domain"
)

type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

func (r *AuditRepo) Create(ctx context.Context, entry *domain.AuditEntry) error {
	detailsJSON, err := json.Marshal(entry.Details)
	if err != nil {
		return fmt.Errorf("marshal details: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO audit_log (actor, actor_type, action, resource, resource_id, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, entry.Actor, entry.ActorType, entry.Action, entry.Resource,
		entry.ResourceID, detailsJSON, entry.IPAddress).
		Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}
	return nil
}

func (r *AuditRepo) List(ctx context.Context, f domain.AuditFilter) ([]*domain.AuditEntry, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}
	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if f.Actor != nil {
		where += fmt.Sprintf(" AND actor = $%d", argIdx)
		args = append(args, *f.Actor)
		argIdx++
	}
	if f.Action != nil {
		where += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, *f.Action)
		argIdx++
	}
	if f.Resource != nil {
		where += fmt.Sprintf(" AND resource = $%d", argIdx)
		args = append(args, *f.Resource)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM audit_log " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit entries: %w", err)
	}

	orderDir := "DESC"
	if f.SortOrder == "asc" {
		orderDir = "ASC"
	}

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`
		SELECT id, actor, actor_type, action, resource, resource_id, details, ip_address, created_at
		FROM audit_log %s
		ORDER BY created_at %s
		LIMIT $%d OFFSET $%d
	`, where, orderDir, argIdx, argIdx+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit entries: %w", err)
	}
	defer rows.Close()

	var entries []*domain.AuditEntry
	for rows.Next() {
		e := &domain.AuditEntry{}
		var detailsJSON []byte
		if err := rows.Scan(
			&e.ID, &e.Actor, &e.ActorType, &e.Action, &e.Resource,
			&e.ResourceID, &detailsJSON, &e.IPAddress, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit entry: %w", err)
		}
		if err := json.Unmarshal(detailsJSON, &e.Details); err != nil {
			e.Details = map[string]interface{}{}
		}
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []*domain.AuditEntry{}
	}

	return entries, total, nil
}
