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

type DeploymentRepo struct {
	pool *pgxpool.Pool
}

func NewDeploymentRepo(pool *pgxpool.Pool) *DeploymentRepo {
	return &DeploymentRepo{pool: pool}
}

func (r *DeploymentRepo) Create(ctx context.Context, d *domain.Deployment) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO deployments (
			name, artifact_id, status, target_device_ids,
			target_device_tags, target_device_types, max_parallel
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, created_at
	`,
		d.Name, d.ArtifactID, d.Status, d.TargetDeviceIDs,
		d.TargetDeviceTags, d.TargetDeviceTypes, d.MaxParallel,
	).Scan(&d.ID, &d.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert deployment: %w", err)
	}
	return nil
}

func (r *DeploymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deployment, error) {
	d := &domain.Deployment{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, artifact_id, status, target_device_ids,
		       target_device_tags, target_device_types, max_parallel,
		       created_at, started_at, finished_at
		FROM deployments WHERE id = $1
	`, id).Scan(
		&d.ID, &d.Name, &d.ArtifactID, &d.Status, &d.TargetDeviceIDs,
		&d.TargetDeviceTags, &d.TargetDeviceTypes, &d.MaxParallel,
		&d.CreatedAt, &d.StartedAt, &d.FinishedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get deployment: %w", err)
	}
	return d, nil
}

func (r *DeploymentRepo) List(ctx context.Context, f domain.DeploymentFilter) ([]*domain.Deployment, int, error) {
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

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM deployments "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deployments: %w", err)
	}

	orderCol := "created_at"
	switch f.SortBy {
	case "created_at", "started_at", "finished_at", "status", "name":
		orderCol = f.SortBy
	}
	orderDir := "DESC"
	if f.SortOrder == "asc" {
		orderDir = "ASC"
	}

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`
		SELECT id, name, artifact_id, status, target_device_ids,
		       target_device_tags, target_device_types, max_parallel,
		       created_at, started_at, finished_at
		FROM deployments %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, orderCol, orderDir, argIdx, argIdx+1)
	args = append(args, f.PerPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []*domain.Deployment
	for rows.Next() {
		d := &domain.Deployment{}
		if err := rows.Scan(
			&d.ID, &d.Name, &d.ArtifactID, &d.Status, &d.TargetDeviceIDs,
			&d.TargetDeviceTags, &d.TargetDeviceTypes, &d.MaxParallel,
			&d.CreatedAt, &d.StartedAt, &d.FinishedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan deployment: %w", err)
		}
		deployments = append(deployments, d)
	}

	if deployments == nil {
		deployments = []*domain.Deployment{}
	}

	return deployments, total, nil
}

func (r *DeploymentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeploymentStatus) error {
	tag, err := r.pool.Exec(ctx, `UPDATE deployments SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("update deployment status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeploymentRepo) SetStarted(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE deployments SET started_at = NOW(), status = 'active' WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("set deployment started: %w", err)
	}
	return nil
}

func (r *DeploymentRepo) SetFinished(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE deployments SET finished_at = NOW(), status = 'completed' WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("set deployment finished: %w", err)
	}
	return nil
}

func (r *DeploymentRepo) GetStats(ctx context.Context) (*domain.DeploymentStats, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM deployments GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	defer rows.Close()

	stats := &domain.DeploymentStats{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats.Total += count
		switch domain.DeploymentStatus(status) {
		case domain.DeploymentStatusScheduled:
			stats.Scheduled = count
		case domain.DeploymentStatusActive:
			stats.Active = count
		case domain.DeploymentStatusCompleted:
			stats.Completed = count
		case domain.DeploymentStatusCancelled:
			stats.Cancelled = count
		}
	}
	return stats, nil
}

// DeploymentDevice operations

func (r *DeploymentRepo) CreateDeploymentDevice(ctx context.Context, dd *domain.DeploymentDevice) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO deployment_devices (deployment_id, device_id, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`, dd.DeploymentID, dd.DeviceID, dd.Status).Scan(&dd.ID)

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return fmt.Errorf("insert deployment_device: %w", err)
	}
	return nil
}

func (r *DeploymentRepo) GetDeploymentDevices(ctx context.Context, deploymentID uuid.UUID) ([]*domain.DeploymentDevice, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, deployment_id, device_id, status, attempts, log, started_at, finished_at
		FROM deployment_devices WHERE deployment_id = $1
		ORDER BY device_id
	`, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("get deployment devices: %w", err)
	}
	defer rows.Close()

	var items []*domain.DeploymentDevice
	for rows.Next() {
		dd := &domain.DeploymentDevice{}
		if err := rows.Scan(
			&dd.ID, &dd.DeploymentID, &dd.DeviceID, &dd.Status,
			&dd.Attempts, &dd.Log, &dd.StartedAt, &dd.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan deployment_device: %w", err)
		}
		items = append(items, dd)
	}

	if items == nil {
		items = []*domain.DeploymentDevice{}
	}

	return items, nil
}

func (r *DeploymentRepo) GetPendingDeploymentForDevice(ctx context.Context, deviceID uuid.UUID) (*domain.DeploymentDevice, *domain.Deployment, *domain.Artifact, error) {
	dd := &domain.DeploymentDevice{}
	dep := &domain.Deployment{}
	art := &domain.Artifact{}

	err := r.pool.QueryRow(ctx, `
		SELECT
			dd.id, dd.deployment_id, dd.device_id, dd.status, dd.attempts,
			d.id, d.name, d.artifact_id, d.status,
			a.id, a.name, a.version, a.file_name, a.file_size, a.checksum_sha256,
			a.target_path, a.file_mode, a.file_owner, a.device_types, a.storage_path,
			a.pre_install_cmd, a.post_install_cmd, a.rollback_cmd
		FROM deployment_devices dd
		JOIN deployments d ON d.id = dd.deployment_id
		JOIN artifacts a ON a.id = d.artifact_id
		WHERE dd.device_id = $1
		  AND dd.status = 'pending'
		  AND d.status IN ('scheduled', 'active')
		ORDER BY d.created_at ASC
		LIMIT 1
	`, deviceID).Scan(
		&dd.ID, &dd.DeploymentID, &dd.DeviceID, &dd.Status, &dd.Attempts,
		&dep.ID, &dep.Name, &dep.ArtifactID, &dep.Status,
		&art.ID, &art.Name, &art.Version, &art.FileName, &art.FileSize,
		&art.ChecksumSHA256, &art.TargetPath, &art.FileMode, &art.FileOwner,
		&art.DeviceTypes, &art.StoragePath, &art.PreInstallCmd, &art.PostInstallCmd,
		&art.RollbackCmd,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil, domain.ErrNotFound
		}
		return nil, nil, nil, fmt.Errorf("get pending deployment: %w", err)
	}

	return dd, dep, art, nil
}

func (r *DeploymentRepo) UpdateDeploymentDeviceStatus(ctx context.Context, id uuid.UUID, status domain.DeploymentDeviceStatus, log string) error {
	var query string
	switch status {
	case domain.DDStatusDownloading, domain.DDStatusInstalling:
		query = `UPDATE deployment_devices SET status = $1, log = $2, attempts = attempts + 1, started_at = COALESCE(started_at, NOW()) WHERE id = $3`
	case domain.DDStatusSuccess, domain.DDStatusFailure:
		query = `UPDATE deployment_devices SET status = $1, log = $2, finished_at = NOW() WHERE id = $3`
	default:
		query = `UPDATE deployment_devices SET status = $1, log = $2 WHERE id = $3`
	}

	tag, err := r.pool.Exec(ctx, query, status, log, id)
	if err != nil {
		return fmt.Errorf("update dd status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeploymentRepo) CountDeploymentDevicesByStatus(ctx context.Context, deploymentID uuid.UUID) (map[domain.DeploymentDeviceStatus]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM deployment_devices
		WHERE deployment_id = $1 GROUP BY status
	`, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("count dd by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[domain.DeploymentDeviceStatus]int)
	for rows.Next() {
		var status domain.DeploymentDeviceStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan dd count: %w", err)
		}
		counts[status] = count
	}
	return counts, nil
}

// suppress unused import
var _ = json.Marshal
