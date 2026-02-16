package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CaioWing/Harbor/internal/domain"
)

type ArtifactRepo struct {
	pool *pgxpool.Pool
}

func NewArtifactRepo(pool *pgxpool.Pool) *ArtifactRepo {
	return &ArtifactRepo{pool: pool}
}

func (r *ArtifactRepo) Create(ctx context.Context, a *domain.Artifact) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO artifacts (
			name, version, description, file_name, file_size, checksum_sha256,
			target_path, file_mode, file_owner, device_types, storage_path,
			pre_install_cmd, post_install_cmd, rollback_cmd
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at
	`,
		a.Name, a.Version, a.Description, a.FileName, a.FileSize, a.ChecksumSHA256,
		a.TargetPath, a.FileMode, a.FileOwner, a.DeviceTypes, a.StoragePath,
		a.PreInstallCmd, a.PostInstallCmd, a.RollbackCmd,
	).Scan(&a.ID, &a.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("insert artifact: %w", err)
	}
	return nil
}

func (r *ArtifactRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error) {
	a := &domain.Artifact{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, version, description, file_name, file_size, checksum_sha256,
		       target_path, file_mode, file_owner, device_types, storage_path,
		       pre_install_cmd, post_install_cmd, rollback_cmd, created_at
		FROM artifacts WHERE id = $1
	`, id).Scan(
		&a.ID, &a.Name, &a.Version, &a.Description, &a.FileName, &a.FileSize,
		&a.ChecksumSHA256, &a.TargetPath, &a.FileMode, &a.FileOwner, &a.DeviceTypes,
		&a.StoragePath, &a.PreInstallCmd, &a.PostInstallCmd, &a.RollbackCmd, &a.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get artifact: %w", err)
	}
	return a, nil
}

func (r *ArtifactRepo) List(ctx context.Context, f domain.ArtifactFilter) ([]*domain.Artifact, int, error) {
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

	if f.Name != nil {
		where += fmt.Sprintf(" AND name ILIKE $%d", argIdx)
		args = append(args, "%"+*f.Name+"%")
		argIdx++
	}
	if f.DeviceType != nil {
		where += fmt.Sprintf(" AND $%d = ANY(device_types)", argIdx)
		args = append(args, *f.DeviceType)
		argIdx++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM artifacts "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count artifacts: %w", err)
	}

	orderCol := "created_at"
	switch f.SortBy {
	case "created_at", "name", "version", "file_size":
		orderCol = f.SortBy
	}
	orderDir := "DESC"
	if f.SortOrder == "asc" {
		orderDir = "ASC"
	}

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`
		SELECT id, name, version, description, file_name, file_size, checksum_sha256,
		       target_path, file_mode, file_owner, device_types, storage_path,
		       pre_install_cmd, post_install_cmd, rollback_cmd, created_at
		FROM artifacts %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, orderCol, orderDir, argIdx, argIdx+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []*domain.Artifact
	for rows.Next() {
		a := &domain.Artifact{}
		if err := rows.Scan(
			&a.ID, &a.Name, &a.Version, &a.Description, &a.FileName, &a.FileSize,
			&a.ChecksumSHA256, &a.TargetPath, &a.FileMode, &a.FileOwner, &a.DeviceTypes,
			&a.StoragePath, &a.PreInstallCmd, &a.PostInstallCmd, &a.RollbackCmd, &a.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan artifact: %w", err)
		}
		artifacts = append(artifacts, a)
	}

	if artifacts == nil {
		artifacts = []*domain.Artifact{}
	}

	return artifacts, total, nil
}

func (r *ArtifactRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM artifacts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
