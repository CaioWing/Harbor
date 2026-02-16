package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DeploymentStatus string

const (
	DeploymentStatusScheduled DeploymentStatus = "scheduled"
	DeploymentStatusActive    DeploymentStatus = "active"
	DeploymentStatusCompleted DeploymentStatus = "completed"
	DeploymentStatusCancelled DeploymentStatus = "cancelled"
)

type DeploymentDeviceStatus string

const (
	DDStatusPending     DeploymentDeviceStatus = "pending"
	DDStatusDownloading DeploymentDeviceStatus = "downloading"
	DDStatusInstalling  DeploymentDeviceStatus = "installing"
	DDStatusSuccess     DeploymentDeviceStatus = "success"
	DDStatusFailure     DeploymentDeviceStatus = "failure"
	DDStatusSkipped     DeploymentDeviceStatus = "skipped"
)

type Deployment struct {
	ID               uuid.UUID        `json:"id"`
	Name             string           `json:"name"`
	ArtifactID       uuid.UUID        `json:"artifact_id"`
	Status           DeploymentStatus `json:"status"`
	TargetDeviceIDs  []uuid.UUID      `json:"target_device_ids,omitempty"`
	TargetDeviceTags []string         `json:"target_device_tags,omitempty"`
	TargetDeviceTypes []string        `json:"target_device_types,omitempty"`
	MaxParallel      int              `json:"max_parallel"`
	CreatedAt        time.Time        `json:"created_at"`
	StartedAt        *time.Time       `json:"started_at,omitempty"`
	FinishedAt       *time.Time       `json:"finished_at,omitempty"`
}

type DeploymentDevice struct {
	ID           uuid.UUID              `json:"id"`
	DeploymentID uuid.UUID              `json:"deployment_id"`
	DeviceID     uuid.UUID              `json:"device_id"`
	Status       DeploymentDeviceStatus `json:"status"`
	Attempts     int                    `json:"attempts"`
	Log          string                 `json:"log"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	FinishedAt   *time.Time             `json:"finished_at,omitempty"`
}

type DeploymentFilter struct {
	Status    *DeploymentStatus
	Page      int
	PerPage   int
	SortBy    string
	SortOrder string
}

type DeploymentStats struct {
	Total     int `json:"total"`
	Scheduled int `json:"scheduled"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Cancelled int `json:"cancelled"`
}

type DeploymentRepository interface {
	Create(ctx context.Context, deployment *Deployment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Deployment, error)
	List(ctx context.Context, filter DeploymentFilter) ([]*Deployment, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status DeploymentStatus) error
	SetStarted(ctx context.Context, id uuid.UUID) error
	SetFinished(ctx context.Context, id uuid.UUID) error
	GetStats(ctx context.Context) (*DeploymentStats, error)

	// DeploymentDevice operations
	CreateDeploymentDevice(ctx context.Context, dd *DeploymentDevice) error
	GetDeploymentDevices(ctx context.Context, deploymentID uuid.UUID) ([]*DeploymentDevice, error)
	GetPendingDeploymentForDevice(ctx context.Context, deviceID uuid.UUID) (*DeploymentDevice, *Deployment, *Artifact, error)
	UpdateDeploymentDeviceStatus(ctx context.Context, id uuid.UUID, status DeploymentDeviceStatus, log string) error
	CountDeploymentDevicesByStatus(ctx context.Context, deploymentID uuid.UUID) (map[DeploymentDeviceStatus]int, error)
}
