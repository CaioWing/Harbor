package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DeviceStatus string

const (
	DeviceStatusPending        DeviceStatus = "pending"
	DeviceStatusAccepted       DeviceStatus = "accepted"
	DeviceStatusRejected       DeviceStatus = "rejected"
	DeviceStatusDecommissioned DeviceStatus = "decommissioned"
)

type IdentityData map[string]string

type Device struct {
	ID            uuid.UUID              `json:"id"`
	IdentityHash  string                 `json:"identity_hash"`
	IdentityData  IdentityData           `json:"identity_data"`
	Status        DeviceStatus           `json:"status"`
	AuthTokenHash string                 `json:"-"`
	Inventory     map[string]interface{} `json:"inventory"`
	DeviceType    string                 `json:"device_type"`
	Tags          []string               `json:"tags"`
	LastCheckIn   *time.Time             `json:"last_check_in"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type DeviceFilter struct {
	Status     *DeviceStatus
	DeviceType *string
	Tags       []string
	Page       int
	PerPage    int
	SortBy     string
	SortOrder  string
}

type DeviceRepository interface {
	Create(ctx context.Context, device *Device) error
	GetByID(ctx context.Context, id uuid.UUID) (*Device, error)
	GetByIdentityHash(ctx context.Context, hash string) (*Device, error)
	List(ctx context.Context, filter DeviceFilter) ([]*Device, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status DeviceStatus) error
	UpdateAuthToken(ctx context.Context, id uuid.UUID, tokenHash string) error
	UpdateInventory(ctx context.Context, id uuid.UUID, inventory map[string]interface{}) error
	UpdateTags(ctx context.Context, id uuid.UUID, tags []string) error
	UpdateLastCheckIn(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByStatus(ctx context.Context) (map[DeviceStatus]int, error)
}
