package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/storage"
)

type ArtifactService struct {
	repo  domain.ArtifactRepository
	store storage.FileStore
	log   *slog.Logger
}

func NewArtifactService(repo domain.ArtifactRepository, store storage.FileStore, log *slog.Logger) *ArtifactService {
	return &ArtifactService{repo: repo, store: store, log: log}
}

type CreateArtifactInput struct {
	Name           string
	Version        string
	Description    string
	FileName       string
	TargetPath     string
	FileMode       string
	FileOwner      string
	DeviceTypes    []string
	PreInstallCmd  string
	PostInstallCmd string
	RollbackCmd    string
	File           io.Reader
}

func (s *ArtifactService) Create(ctx context.Context, input CreateArtifactInput) (*domain.Artifact, error) {
	if input.Name == "" || input.Version == "" || input.TargetPath == "" {
		return nil, fmt.Errorf("%w: name, version, and target_path are required", domain.ErrInvalidInput)
	}
	if len(input.DeviceTypes) == 0 {
		return nil, fmt.Errorf("%w: at least one device_type is required", domain.ErrInvalidInput)
	}
	if input.FileMode == "" {
		input.FileMode = "0644"
	}

	// Hash the file while saving
	hasher := sha256.New()
	tee := io.TeeReader(input.File, hasher)

	storageName := fmt.Sprintf("%s_%s_%s", input.Name, input.Version, input.FileName)
	storagePath, fileSize, err := s.store.Save(storageName, tee)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	artifact := &domain.Artifact{
		Name:           input.Name,
		Version:        input.Version,
		Description:    input.Description,
		FileName:       input.FileName,
		FileSize:       fileSize,
		ChecksumSHA256: checksum,
		TargetPath:     input.TargetPath,
		FileMode:       input.FileMode,
		FileOwner:      input.FileOwner,
		DeviceTypes:    input.DeviceTypes,
		StoragePath:    storagePath,
		PreInstallCmd:  input.PreInstallCmd,
		PostInstallCmd: input.PostInstallCmd,
		RollbackCmd:    input.RollbackCmd,
	}

	if err := s.repo.Create(ctx, artifact); err != nil {
		// Clean up stored file on DB error
		s.store.Delete(storagePath)
		return nil, fmt.Errorf("create artifact: %w", err)
	}

	s.log.Info("artifact created", "id", artifact.ID, "name", artifact.Name, "version", artifact.Version)
	return artifact, nil
}

func (s *ArtifactService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Artifact, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ArtifactService) List(ctx context.Context, filter domain.ArtifactFilter) ([]*domain.Artifact, int, error) {
	return s.repo.List(ctx, filter)
}

func (s *ArtifactService) OpenFile(ctx context.Context, id uuid.UUID) (io.ReadCloser, *domain.Artifact, error) {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.store.Open(artifact.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open artifact file: %w", err)
	}

	return reader, artifact, nil
}

func (s *ArtifactService) Delete(ctx context.Context, id uuid.UUID) error {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if err := s.store.Delete(artifact.StoragePath); err != nil {
		s.log.Warn("failed to delete artifact file", "path", artifact.StoragePath, "err", err)
	}

	s.log.Info("artifact deleted", "id", id)
	return nil
}
