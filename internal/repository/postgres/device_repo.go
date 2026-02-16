package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CaioWing/Harbor/internal/domain"
)

type DeviceRepo struct {
	pool *pgxpool.Pool
}

func NewDeviceRepo(pool *pgxpool.Pool) *DeviceRepo {
	return &DeviceRepo{pool: pool}
}

func (r *DeviceRepo) Create(ctx context.Context, d *domain.Device) error {
	identityJSON, err := json.Marshal(d.IdentityData)
	if err != nil {
		return fmt.Errorf("marshal identity: %w", err)
	}
	inventoryJSON, err := json.Marshal(d.Inventory)
	if err != nil {
		return fmt.Errorf("marshal inventory: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO devices (identity_hash, identity_data, status, device_type, inventory, tags)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, d.IdentityHash, identityJSON, d.Status, d.DeviceType, inventoryJSON, d.Tags).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("insert device: %w", err)
	}
	return nil
}

func (r *DeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	d := &domain.Device{}
	var identityJSON, inventoryJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, identity_hash, identity_data, status, auth_token_hash,
		       inventory, device_type, tags, last_check_in, created_at, updated_at
		FROM devices WHERE id = $1
	`, id).Scan(
		&d.ID, &d.IdentityHash, &identityJSON, &d.Status, &d.AuthTokenHash,
		&inventoryJSON, &d.DeviceType, &d.Tags, &d.LastCheckIn, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get device: %w", err)
	}

	if err := json.Unmarshal(identityJSON, &d.IdentityData); err != nil {
		return nil, fmt.Errorf("unmarshal identity: %w", err)
	}
	if err := json.Unmarshal(inventoryJSON, &d.Inventory); err != nil {
		return nil, fmt.Errorf("unmarshal inventory: %w", err)
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}

	return d, nil
}

func (r *DeviceRepo) GetByIdentityHash(ctx context.Context, hash string) (*domain.Device, error) {
	d := &domain.Device{}
	var identityJSON, inventoryJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, identity_hash, identity_data, status, auth_token_hash,
		       inventory, device_type, tags, last_check_in, created_at, updated_at
		FROM devices WHERE identity_hash = $1
	`, hash).Scan(
		&d.ID, &d.IdentityHash, &identityJSON, &d.Status, &d.AuthTokenHash,
		&inventoryJSON, &d.DeviceType, &d.Tags, &d.LastCheckIn, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get device by hash: %w", err)
	}

	if err := json.Unmarshal(identityJSON, &d.IdentityData); err != nil {
		return nil, fmt.Errorf("unmarshal identity: %w", err)
	}
	if err := json.Unmarshal(inventoryJSON, &d.Inventory); err != nil {
		return nil, fmt.Errorf("unmarshal inventory: %w", err)
	}
	if d.Tags == nil {
		d.Tags = []string{}
	}

	return d, nil
}

func (r *DeviceRepo) List(ctx context.Context, f domain.DeviceFilter) ([]*domain.Device, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}
	if f.SortBy == "" {
		f.SortBy = "created_at"
	}
	if f.SortOrder == "" {
		f.SortOrder = "desc"
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if f.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *f.Status)
		argIdx++
	}
	if f.DeviceType != nil {
		where += fmt.Sprintf(" AND device_type = $%d", argIdx)
		args = append(args, *f.DeviceType)
		argIdx++
	}
	if len(f.Tags) > 0 {
		where += fmt.Sprintf(" AND tags @> $%d", argIdx)
		args = append(args, f.Tags)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM devices " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count devices: %w", err)
	}

	orderCol := "created_at"
	switch f.SortBy {
	case "created_at", "updated_at", "device_type", "status", "last_check_in":
		orderCol = f.SortBy
	}
	orderDir := "DESC"
	if f.SortOrder == "asc" {
		orderDir = "ASC"
	}

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`
		SELECT id, identity_hash, identity_data, status, inventory,
		       device_type, tags, last_check_in, created_at, updated_at
		FROM devices %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, orderCol, orderDir, argIdx, argIdx+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []*domain.Device
	for rows.Next() {
		d := &domain.Device{}
		var identityJSON, inventoryJSON []byte
		if err := rows.Scan(
			&d.ID, &d.IdentityHash, &identityJSON, &d.Status, &inventoryJSON,
			&d.DeviceType, &d.Tags, &d.LastCheckIn, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan device: %w", err)
		}
		json.Unmarshal(identityJSON, &d.IdentityData)
		json.Unmarshal(inventoryJSON, &d.Inventory)
		if d.Tags == nil {
			d.Tags = []string{}
		}
		devices = append(devices, d)
	}

	if devices == nil {
		devices = []*domain.Device{}
	}

	return devices, total, nil
}

func (r *DeviceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeviceStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE devices SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("update device status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeviceRepo) UpdateAuthToken(ctx context.Context, id uuid.UUID, tokenHash string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE devices SET auth_token_hash = $1, updated_at = NOW() WHERE id = $2
	`, tokenHash, id)
	if err != nil {
		return fmt.Errorf("update auth token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeviceRepo) UpdateInventory(ctx context.Context, id uuid.UUID, inventory map[string]interface{}) error {
	inventoryJSON, err := json.Marshal(inventory)
	if err != nil {
		return fmt.Errorf("marshal inventory: %w", err)
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE devices SET inventory = $1, updated_at = NOW() WHERE id = $2
	`, inventoryJSON, id)
	if err != nil {
		return fmt.Errorf("update inventory: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeviceRepo) UpdateTags(ctx context.Context, id uuid.UUID, tags []string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE devices SET tags = $1, updated_at = NOW() WHERE id = $2
	`, tags, id)
	if err != nil {
		return fmt.Errorf("update tags: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeviceRepo) UpdateLastCheckIn(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE devices SET last_check_in = NOW(), updated_at = NOW() WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("update last check-in: %w", err)
	}
	return nil
}

func (r *DeviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE devices SET status = 'decommissioned', updated_at = NOW() WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("decommission device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeviceRepo) CountByStatus(ctx context.Context) (map[domain.DeviceStatus]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM devices GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[domain.DeviceStatus]int)
	for rows.Next() {
		var status domain.DeviceStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		counts[status] = count
	}
	return counts, nil
}
