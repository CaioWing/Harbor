package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Artifact struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Version        string    `json:"version"`
	Description    string    `json:"description"`
	FileName       string    `json:"file_name"`
	FileSize       int64     `json:"file_size"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
	TargetPath     string    `json:"target_path"`
	FileMode       string    `json:"file_mode"`
	FileOwner      string    `json:"file_owner"`
	DeviceTypes    []string  `json:"device_types"`
	StoragePath    string    `json:"-"`
	PreInstallCmd  string    `json:"pre_install_cmd,omitempty"`
	PostInstallCmd string    `json:"post_install_cmd,omitempty"`
	RollbackCmd    string    `json:"rollback_cmd,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type ArtifactFilter struct {
	Name       *string
	DeviceType *string
	Page       int
	PerPage    int
	SortBy     string
	SortOrder  string
}

type ArtifactRepository interface {
	Create(ctx context.Context, artifact *Artifact) error
	GetByID(ctx context.Context, id uuid.UUID) (*Artifact, error)
	List(ctx context.Context, filter ArtifactFilter) ([]*Artifact, int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
