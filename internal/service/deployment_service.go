package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/domain"
)

type DeploymentService struct {
	deployRepo domain.DeploymentRepository
	deviceRepo domain.DeviceRepository
	artRepo    domain.ArtifactRepository
	log        *slog.Logger
}

func NewDeploymentService(
	deployRepo domain.DeploymentRepository,
	deviceRepo domain.DeviceRepository,
	artRepo domain.ArtifactRepository,
	log *slog.Logger,
) *DeploymentService {
	return &DeploymentService{
		deployRepo: deployRepo,
		deviceRepo: deviceRepo,
		artRepo:    artRepo,
		log:        log,
	}
}

type CreateDeploymentInput struct {
	Name              string
	ArtifactID        uuid.UUID
	TargetDeviceIDs   []uuid.UUID
	TargetDeviceTags  []string
	TargetDeviceTypes []string
	MaxParallel       int
}

func (s *DeploymentService) Create(ctx context.Context, input CreateDeploymentInput) (*domain.Deployment, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}

	// Verify artifact exists
	artifact, err := s.artRepo.GetByID(ctx, input.ArtifactID)
	if err != nil {
		return nil, fmt.Errorf("artifact: %w", err)
	}

	// Resolve target devices
	deviceIDs, err := s.resolveTargets(ctx, input, artifact)
	if err != nil {
		return nil, fmt.Errorf("resolve targets: %w", err)
	}

	if len(deviceIDs) == 0 {
		return nil, fmt.Errorf("%w: no matching devices found", domain.ErrInvalidInput)
	}

	deployment := &domain.Deployment{
		Name:              input.Name,
		ArtifactID:        input.ArtifactID,
		Status:            domain.DeploymentStatusScheduled,
		TargetDeviceIDs:   input.TargetDeviceIDs,
		TargetDeviceTags:  input.TargetDeviceTags,
		TargetDeviceTypes: input.TargetDeviceTypes,
		MaxParallel:       input.MaxParallel,
	}

	if err := s.deployRepo.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("create deployment: %w", err)
	}

	// Create deployment_device entries for each target
	for _, deviceID := range deviceIDs {
		dd := &domain.DeploymentDevice{
			DeploymentID: deployment.ID,
			DeviceID:     deviceID,
			Status:       domain.DDStatusPending,
		}
		if err := s.deployRepo.CreateDeploymentDevice(ctx, dd); err != nil {
			if errors.Is(err, domain.ErrConflict) {
				continue // skip duplicates
			}
			s.log.Warn("failed to create deployment_device", "device", deviceID, "err", err)
		}
	}

	// Activate deployment
	if err := s.deployRepo.SetStarted(ctx, deployment.ID); err != nil {
		s.log.Warn("failed to activate deployment", "id", deployment.ID, "err", err)
	}

	s.log.Info("deployment created", "id", deployment.ID, "devices", len(deviceIDs))
	return deployment, nil
}

func (s *DeploymentService) resolveTargets(ctx context.Context, input CreateDeploymentInput, artifact *domain.Artifact) ([]uuid.UUID, error) {
	deviceIDSet := make(map[uuid.UUID]bool)

	// Add explicitly targeted devices
	for _, id := range input.TargetDeviceIDs {
		device, err := s.deviceRepo.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if device.Status == domain.DeviceStatusAccepted {
			deviceIDSet[id] = true
		}
	}

	// Resolve by tags
	if len(input.TargetDeviceTags) > 0 {
		devices, _, err := s.deviceRepo.List(ctx, domain.DeviceFilter{
			Status:  statusPtr(domain.DeviceStatusAccepted),
			Tags:    input.TargetDeviceTags,
			Page:    1,
			PerPage: 100,
		})
		if err != nil {
			return nil, err
		}
		for _, d := range devices {
			deviceIDSet[d.ID] = true
		}
	}

	// Resolve by device types
	targetTypes := input.TargetDeviceTypes
	if len(targetTypes) == 0 {
		targetTypes = artifact.DeviceTypes
	}
	for _, dt := range targetTypes {
		devices, _, err := s.deviceRepo.List(ctx, domain.DeviceFilter{
			Status:     statusPtr(domain.DeviceStatusAccepted),
			DeviceType: &dt,
			Page:       1,
			PerPage:    100,
		})
		if err != nil {
			return nil, err
		}
		for _, d := range devices {
			deviceIDSet[d.ID] = true
		}
	}

	ids := make([]uuid.UUID, 0, len(deviceIDSet))
	for id := range deviceIDSet {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *DeploymentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deployment, error) {
	return s.deployRepo.GetByID(ctx, id)
}

func (s *DeploymentService) List(ctx context.Context, filter domain.DeploymentFilter) ([]*domain.Deployment, int, error) {
	return s.deployRepo.List(ctx, filter)
}

func (s *DeploymentService) Cancel(ctx context.Context, id uuid.UUID) error {
	dep, err := s.deployRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if dep.Status == domain.DeploymentStatusCompleted || dep.Status == domain.DeploymentStatusCancelled {
		return fmt.Errorf("%w: deployment already %s", domain.ErrInvalidInput, dep.Status)
	}

	// Skip remaining pending devices
	devices, err := s.deployRepo.GetDeploymentDevices(ctx, id)
	if err != nil {
		return err
	}
	for _, dd := range devices {
		if dd.Status == domain.DDStatusPending {
			s.deployRepo.UpdateDeploymentDeviceStatus(ctx, dd.ID, domain.DDStatusSkipped, "deployment cancelled")
		}
	}

	return s.deployRepo.UpdateStatus(ctx, id, domain.DeploymentStatusCancelled)
}

func (s *DeploymentService) GetDeploymentDevices(ctx context.Context, deploymentID uuid.UUID) ([]*domain.DeploymentDevice, error) {
	return s.deployRepo.GetDeploymentDevices(ctx, deploymentID)
}

func (s *DeploymentService) GetNextForDevice(ctx context.Context, deviceID uuid.UUID) (*domain.DeploymentDevice, *domain.Deployment, *domain.Artifact, error) {
	return s.deployRepo.GetPendingDeploymentForDevice(ctx, deviceID)
}

func (s *DeploymentService) UpdateDeviceStatus(ctx context.Context, ddID uuid.UUID, status domain.DeploymentDeviceStatus, log string) error {
	if err := s.deployRepo.UpdateDeploymentDeviceStatus(ctx, ddID, status, log); err != nil {
		return err
	}

	// Check if deployment is fully complete after a terminal status
	if status == domain.DDStatusSuccess || status == domain.DDStatusFailure {
		s.checkDeploymentCompletion(ctx, ddID)
	}

	return nil
}

func (s *DeploymentService) checkDeploymentCompletion(ctx context.Context, ddID uuid.UUID) {
	// This is a simplification — in production, get deploymentID from ddID
	// For now, this is best-effort
}

func (s *DeploymentService) GetStats(ctx context.Context) (*domain.DeploymentStats, error) {
	return s.deployRepo.GetStats(ctx)
}
