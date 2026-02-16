package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/domain"
)

type deploymentTestEnv struct {
	svc        *DeploymentService
	deployRepo *mockDeploymentRepo
	deviceRepo *mockDeviceRepo
	artRepo    *mockArtifactRepo
}

func newTestDeploymentService() *deploymentTestEnv {
	deviceRepo := newMockDeviceRepo()
	artRepo := newMockArtifactRepo()
	deployRepo := newMockDeploymentRepo(artRepo)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewDeploymentService(deployRepo, deviceRepo, artRepo, log)
	return &deploymentTestEnv{
		svc:        svc,
		deployRepo: deployRepo,
		deviceRepo: deviceRepo,
		artRepo:    artRepo,
	}
}

func (e *deploymentTestEnv) createAcceptedDevice(ctx context.Context, deviceType string, tags []string) *domain.Device {
	d := &domain.Device{
		IdentityHash: uuid.New().String(),
		IdentityData: domain.IdentityData{"device_type": deviceType},
		Status:       domain.DeviceStatusAccepted,
		DeviceType:   deviceType,
		Inventory:    map[string]interface{}{},
		Tags:         tags,
	}
	e.deviceRepo.Create(ctx, d)
	return d
}

func (e *deploymentTestEnv) createArtifact(ctx context.Context, name, version string, deviceTypes []string) *domain.Artifact {
	a := &domain.Artifact{
		Name:        name,
		Version:     version,
		DeviceTypes: deviceTypes,
		TargetPath:  "/usr/local/bin/" + name,
		FileName:    name,
		FileSize:    100,
		StoragePath: "/mock/" + name,
	}
	e.artRepo.Create(ctx, a)
	return a
}

func TestDeploymentCreate_Success(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	device := env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	input := CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	}

	dep, err := env.svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep.Name != "deploy-1" {
		t.Fatalf("expected deploy-1, got %s", dep.Name)
	}
	if dep.ArtifactID != artifact.ID {
		t.Fatal("artifact ID mismatch")
	}
}

func TestDeploymentCreate_MissingName(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	input := CreateDeploymentInput{
		ArtifactID: artifact.ID,
	}

	_, err := env.svc.Create(ctx, input)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeploymentCreate_InvalidArtifact(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	input := CreateDeploymentInput{
		Name:       "deploy-1",
		ArtifactID: uuid.New(),
	}

	_, err := env.svc.Create(ctx, input)
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
}

func TestDeploymentCreate_NoMatchingDevices(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	// Create a device with different type
	env.createAcceptedDevice(ctx, "beaglebone", []string{})

	input := CreateDeploymentInput{
		Name:              "deploy-1",
		ArtifactID:        artifact.ID,
		TargetDeviceTypes: []string{"raspberry-pi-4"},
	}

	_, err := env.svc.Create(ctx, input)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput (no matching devices), got %v", err)
	}
}

func TestDeploymentCreate_ResolvesByTags(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	// Use different device types so the fallback device_type resolution doesn't mix them
	env.createAcceptedDevice(ctx, "type-a", []string{"production"})
	env.createAcceptedDevice(ctx, "type-b", []string{"staging"})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"type-c"}) // no device matches by type

	input := CreateDeploymentInput{
		Name:             "deploy-1",
		ArtifactID:       artifact.ID,
		TargetDeviceTags: []string{"production"},
	}

	dep, err := env.svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 1 deployment device (only production tagged; type-c resolves to 0 extra)
	dds, _ := env.deployRepo.GetDeploymentDevices(ctx, dep.ID)
	if len(dds) != 1 {
		t.Fatalf("expected 1 deployment device, got %d", len(dds))
	}
}

func TestDeploymentCreate_ResolvesByDeviceType(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	env.createAcceptedDevice(ctx, "beaglebone", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	input := CreateDeploymentInput{
		Name:              "deploy-1",
		ArtifactID:        artifact.ID,
		TargetDeviceTypes: []string{"raspberry-pi-4"},
	}

	dep, err := env.svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dds, _ := env.deployRepo.GetDeploymentDevices(ctx, dep.ID)
	if len(dds) != 2 {
		t.Fatalf("expected 2 deployment devices, got %d", len(dds))
	}
}

func TestDeploymentCancel(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	device := env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	dep, _ := env.svc.Create(ctx, CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	})

	err := env.svc.Cancel(ctx, dep.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check deployment status
	updated, _ := env.deployRepo.GetByID(ctx, dep.ID)
	if updated.Status != domain.DeploymentStatusCancelled {
		t.Fatalf("expected cancelled, got %s", updated.Status)
	}

	// Check that pending devices are skipped
	dds, _ := env.deployRepo.GetDeploymentDevices(ctx, dep.ID)
	for _, dd := range dds {
		if dd.Status != domain.DDStatusSkipped {
			t.Fatalf("expected skipped, got %s", dd.Status)
		}
	}
}

func TestDeploymentCancel_AlreadyCancelled(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	device := env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	dep, _ := env.svc.Create(ctx, CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	})

	env.svc.Cancel(ctx, dep.ID)
	err := env.svc.Cancel(ctx, dep.ID)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeploymentGetByID(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	device := env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	dep, _ := env.svc.Create(ctx, CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	})

	found, err := env.svc.GetByID(ctx, dep.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Name != "deploy-1" {
		t.Fatalf("expected deploy-1, got %s", found.Name)
	}
}

func TestDeploymentGetByID_NotFound(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	_, err := env.svc.GetByID(ctx, uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeploymentGetNextForDevice(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	device := env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	env.svc.Create(ctx, CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	})

	dd, dep, art, err := env.svc.GetNextForDevice(ctx, device.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dd == nil || dep == nil || art == nil {
		t.Fatal("expected non-nil results")
	}
	if dd.DeviceID != device.ID {
		t.Fatal("device ID mismatch")
	}
	if dep.Name != "deploy-1" {
		t.Fatalf("expected deploy-1, got %s", dep.Name)
	}
}

func TestDeploymentGetNextForDevice_NoPending(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	_, _, _, err := env.svc.GetNextForDevice(ctx, uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeploymentUpdateDeviceStatus(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	device := env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	dep, _ := env.svc.Create(ctx, CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	})

	dds, _ := env.deployRepo.GetDeploymentDevices(ctx, dep.ID)
	if len(dds) == 0 {
		t.Fatal("expected deployment devices")
	}

	err := env.svc.UpdateDeviceStatus(ctx, dds[0].ID, domain.DDStatusSuccess, "done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, _ := env.deployRepo.GetDeploymentDevices(ctx, dep.ID)
	if updated[0].Status != domain.DDStatusSuccess {
		t.Fatalf("expected success, got %s", updated[0].Status)
	}
}

func TestDeploymentGetStats(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	device := env.createAcceptedDevice(ctx, "raspberry-pi-4", []string{})
	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	// Create and leave active
	env.svc.Create(ctx, CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	})

	// Create and cancel
	dep2, _ := env.svc.Create(ctx, CreateDeploymentInput{
		Name:            "deploy-2",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{device.ID},
	})
	env.svc.Cancel(ctx, dep2.ID)

	stats, err := env.svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Total != 2 {
		t.Fatalf("expected total 2, got %d", stats.Total)
	}
	if stats.Active != 1 {
		t.Fatalf("expected 1 active, got %d", stats.Active)
	}
	if stats.Cancelled != 1 {
		t.Fatalf("expected 1 cancelled, got %d", stats.Cancelled)
	}
}

func TestDeploymentCreate_PendingDeviceExcluded(t *testing.T) {
	env := newTestDeploymentService()
	ctx := context.Background()

	// Create pending device (not accepted)
	d := &domain.Device{
		IdentityHash: uuid.New().String(),
		IdentityData: domain.IdentityData{"device_type": "raspberry-pi-4"},
		Status:       domain.DeviceStatusPending,
		DeviceType:   "raspberry-pi-4",
		Inventory:    map[string]interface{}{},
		Tags:         []string{},
	}
	env.deviceRepo.Create(ctx, d)

	artifact := env.createArtifact(ctx, "myapp", "1.0.0", []string{"raspberry-pi-4"})

	input := CreateDeploymentInput{
		Name:            "deploy-1",
		ArtifactID:      artifact.ID,
		TargetDeviceIDs: []uuid.UUID{d.ID},
	}

	_, err := env.svc.Create(ctx, input)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput (no matching devices), got %v", err)
	}
}
