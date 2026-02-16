package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/auth"
	"github.com/CaioWing/Harbor/internal/domain"
)

func newTestDeviceService() (*DeviceService, *mockDeviceRepo) {
	repo := newMockDeviceRepo()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewDeviceService(repo, log)
	return svc, repo
}

func TestAuthenticate_NewDevice_CreatesPending(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	identity := domain.IdentityData{
		"device_type": "raspberry-pi-4",
		"mac_address": "aa:bb:cc:dd:ee:ff",
	}

	_, err := svc.Authenticate(ctx, identity)
	if !errors.Is(err, domain.ErrDevicePending) {
		t.Fatalf("expected ErrDevicePending, got %v", err)
	}

	// Device should be created in repo
	if len(repo.devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(repo.devices))
	}

	for _, d := range repo.devices {
		if d.Status != domain.DeviceStatusPending {
			t.Fatalf("expected pending status, got %s", d.Status)
		}
		if d.DeviceType != "raspberry-pi-4" {
			t.Fatalf("expected device_type raspberry-pi-4, got %s", d.DeviceType)
		}
	}
}

func TestAuthenticate_MissingDeviceType(t *testing.T) {
	svc, _ := newTestDeviceService()
	ctx := context.Background()

	identity := domain.IdentityData{
		"mac_address": "aa:bb:cc:dd:ee:ff",
	}

	_, err := svc.Authenticate(ctx, identity)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAuthenticate_AcceptedDevice_ReturnsToken(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	identity := domain.IdentityData{
		"device_type": "raspberry-pi-4",
		"mac_address": "aa:bb:cc:dd:ee:ff",
	}

	// First call creates pending device
	svc.Authenticate(ctx, identity)

	// Accept the device
	var deviceID uuid.UUID
	for id := range repo.devices {
		deviceID = id
	}
	repo.UpdateStatus(ctx, deviceID, domain.DeviceStatusAccepted)

	// Second call should return token
	token, err := svc.Authenticate(ctx, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestAuthenticate_RejectedDevice(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	identity := domain.IdentityData{
		"device_type": "raspberry-pi-4",
		"mac_address": "aa:bb:cc:dd:ee:ff",
	}

	svc.Authenticate(ctx, identity)

	var deviceID uuid.UUID
	for id := range repo.devices {
		deviceID = id
	}
	repo.UpdateStatus(ctx, deviceID, domain.DeviceStatusRejected)

	_, err := svc.Authenticate(ctx, identity)
	if !errors.Is(err, domain.ErrDeviceRejected) {
		t.Fatalf("expected ErrDeviceRejected, got %v", err)
	}
}

func TestAuthenticate_DeterministicHash(t *testing.T) {
	svc, _ := newTestDeviceService()
	ctx := context.Background()

	identity := domain.IdentityData{
		"device_type": "raspberry-pi-4",
		"mac_address": "aa:bb:cc:dd:ee:ff",
	}

	// First call
	svc.Authenticate(ctx, identity)

	// Second call with same identity should find the same device
	_, err := svc.Authenticate(ctx, identity)
	if !errors.Is(err, domain.ErrDevicePending) {
		t.Fatalf("expected ErrDevicePending on second call, got %v", err)
	}
}

func TestValidateToken(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	// Create an accepted device with a known token
	token, tokenHash, _ := auth.GenerateDeviceToken()
	device := &domain.Device{
		IdentityHash:  "testhash123",
		IdentityData:  domain.IdentityData{"device_type": "test"},
		Status:        domain.DeviceStatusAccepted,
		AuthTokenHash: tokenHash,
		DeviceType:    "test",
		Inventory:     map[string]interface{}{},
		Tags:          []string{},
	}
	repo.Create(ctx, device)

	// Validate should succeed
	found, err := svc.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != device.ID {
		t.Fatalf("expected device ID %s, got %s", device.ID, found.ID)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	svc, _ := newTestDeviceService()
	ctx := context.Background()

	_, err := svc.ValidateToken(ctx, "invalid-token")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetByID(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	device := &domain.Device{
		IdentityHash: "hash1",
		IdentityData: domain.IdentityData{"device_type": "test"},
		Status:       domain.DeviceStatusPending,
		DeviceType:   "test",
		Inventory:    map[string]interface{}{},
		Tags:         []string{},
	}
	repo.Create(ctx, device)

	found, err := svc.GetByID(ctx, device.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.IdentityHash != "hash1" {
		t.Fatalf("expected hash1, got %s", found.IdentityHash)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _ := newTestDeviceService()
	ctx := context.Background()

	_, err := svc.GetByID(ctx, uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	device := &domain.Device{
		IdentityHash: "hash1",
		IdentityData: domain.IdentityData{"device_type": "test"},
		Status:       domain.DeviceStatusPending,
		DeviceType:   "test",
		Inventory:    map[string]interface{}{},
		Tags:         []string{},
	}
	repo.Create(ctx, device)

	err := svc.UpdateStatus(ctx, device.ID, domain.DeviceStatusAccepted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := repo.GetByID(ctx, device.ID)
	if found.Status != domain.DeviceStatusAccepted {
		t.Fatalf("expected accepted, got %s", found.Status)
	}
}

func TestUpdateInventory(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	device := &domain.Device{
		IdentityHash: "hash1",
		IdentityData: domain.IdentityData{"device_type": "test"},
		Status:       domain.DeviceStatusAccepted,
		DeviceType:   "test",
		Inventory:    map[string]interface{}{},
		Tags:         []string{},
	}
	repo.Create(ctx, device)

	inv := map[string]interface{}{"os": "Linux", "arch": "arm64"}
	err := svc.UpdateInventory(ctx, device.ID, inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := repo.GetByID(ctx, device.ID)
	if found.Inventory["os"] != "Linux" {
		t.Fatal("inventory not updated")
	}
}

func TestUpdateTags(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	device := &domain.Device{
		IdentityHash: "hash1",
		IdentityData: domain.IdentityData{"device_type": "test"},
		Status:       domain.DeviceStatusAccepted,
		DeviceType:   "test",
		Inventory:    map[string]interface{}{},
		Tags:         []string{},
	}
	repo.Create(ctx, device)

	err := svc.UpdateTags(ctx, device.ID, []string{"production", "rack-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := repo.GetByID(ctx, device.ID)
	if len(found.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(found.Tags))
	}
}

func TestDecommission(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	device := &domain.Device{
		IdentityHash: "hash1",
		IdentityData: domain.IdentityData{"device_type": "test"},
		Status:       domain.DeviceStatusAccepted,
		DeviceType:   "test",
		Inventory:    map[string]interface{}{},
		Tags:         []string{},
	}
	repo.Create(ctx, device)

	err := svc.Decommission(ctx, device.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := repo.GetByID(ctx, device.ID)
	if found.Status != domain.DeviceStatusDecommissioned {
		t.Fatalf("expected decommissioned, got %s", found.Status)
	}
}

func TestCountByStatus(t *testing.T) {
	svc, repo := newTestDeviceService()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		repo.Create(ctx, &domain.Device{
			IdentityHash: uuid.New().String(),
			IdentityData: domain.IdentityData{"device_type": "test"},
			Status:       domain.DeviceStatusPending,
			DeviceType:   "test",
			Inventory:    map[string]interface{}{},
			Tags:         []string{},
		})
	}
	repo.Create(ctx, &domain.Device{
		IdentityHash: uuid.New().String(),
		IdentityData: domain.IdentityData{"device_type": "test"},
		Status:       domain.DeviceStatusAccepted,
		DeviceType:   "test",
		Inventory:    map[string]interface{}{},
		Tags:         []string{},
	})

	counts, err := svc.CountByStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if counts[domain.DeviceStatusPending] != 3 {
		t.Fatalf("expected 3 pending, got %d", counts[domain.DeviceStatusPending])
	}
	if counts[domain.DeviceStatusAccepted] != 1 {
		t.Fatalf("expected 1 accepted, got %d", counts[domain.DeviceStatusAccepted])
	}
}
