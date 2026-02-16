package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/auth"
	"github.com/CaioWing/Harbor/internal/domain"
)

type DeviceService struct {
	repo domain.DeviceRepository
	log  *slog.Logger
}

func NewDeviceService(repo domain.DeviceRepository, log *slog.Logger) *DeviceService {
	return &DeviceService{repo: repo, log: log}
}

func (s *DeviceService) Authenticate(ctx context.Context, identityData domain.IdentityData) (string, error) {
	deviceType, ok := identityData["device_type"]
	if !ok || deviceType == "" {
		return "", fmt.Errorf("%w: device_type is required", domain.ErrInvalidInput)
	}

	hash := computeIdentityHash(identityData)

	device, err := s.repo.GetByIdentityHash(ctx, hash)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return "", fmt.Errorf("lookup device: %w", err)
		}

		// New device — create as pending
		device = &domain.Device{
			IdentityHash: hash,
			IdentityData: identityData,
			Status:       domain.DeviceStatusPending,
			DeviceType:   deviceType,
			Inventory:    map[string]interface{}{},
			Tags:         []string{},
		}
		if err := s.repo.Create(ctx, device); err != nil {
			if errors.Is(err, domain.ErrConflict) {
				// Race condition — another request created it
				device, err = s.repo.GetByIdentityHash(ctx, hash)
				if err != nil {
					return "", fmt.Errorf("re-lookup device: %w", err)
				}
			} else {
				return "", fmt.Errorf("create device: %w", err)
			}
		}
		s.log.Info("new device registered", "id", device.ID, "type", deviceType)
	}

	switch device.Status {
	case domain.DeviceStatusPending:
		return "", domain.ErrDevicePending
	case domain.DeviceStatusRejected, domain.DeviceStatusDecommissioned:
		return "", domain.ErrDeviceRejected
	case domain.DeviceStatusAccepted:
		// Generate new token
		token, tokenHash, err := auth.GenerateDeviceToken()
		if err != nil {
			return "", fmt.Errorf("generate token: %w", err)
		}
		if err := s.repo.UpdateAuthToken(ctx, device.ID, tokenHash); err != nil {
			return "", fmt.Errorf("save token: %w", err)
		}
		s.repo.UpdateLastCheckIn(ctx, device.ID)
		s.log.Info("device authenticated", "id", device.ID)
		return token, nil
	default:
		return "", fmt.Errorf("unknown device status: %s", device.Status)
	}
}

func (s *DeviceService) ValidateToken(ctx context.Context, token string) (*domain.Device, error) {
	tokenHash := auth.HashToken(token)
	return s.findByTokenHash(ctx, tokenHash)
}

func (s *DeviceService) findByTokenHash(ctx context.Context, tokenHash string) (*domain.Device, error) {
	// This searches across all devices, which is acceptable for MVP scale.
	// For production, add GetByTokenHash to the repository interface.
	devices, _, err := s.repo.List(ctx, domain.DeviceFilter{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.AuthTokenHash == tokenHash && d.Status == domain.DeviceStatusAccepted {
			return d, nil
		}
	}
	return nil, domain.ErrUnauthorized
}

func (s *DeviceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DeviceService) List(ctx context.Context, filter domain.DeviceFilter) ([]*domain.Device, int, error) {
	return s.repo.List(ctx, filter)
}

func (s *DeviceService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeviceStatus) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *DeviceService) UpdateInventory(ctx context.Context, id uuid.UUID, inventory map[string]interface{}) error {
	return s.repo.UpdateInventory(ctx, id, inventory)
}

func (s *DeviceService) UpdateTags(ctx context.Context, id uuid.UUID, tags []string) error {
	return s.repo.UpdateTags(ctx, id, tags)
}

func (s *DeviceService) Decommission(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *DeviceService) CountByStatus(ctx context.Context) (map[domain.DeviceStatus]int, error) {
	return s.repo.CountByStatus(ctx)
}

func computeIdentityHash(data domain.IdentityData) string {
	// Sort keys for deterministic hashing
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	canonical, _ := json.Marshal(data)
	h := sha256.Sum256(canonical)
	return hex.EncodeToString(h[:])
}

func statusPtr(s domain.DeviceStatus) *domain.DeviceStatus {
	return &s
}
