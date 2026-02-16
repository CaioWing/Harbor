package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/CaioWing/Harbor/internal/domain"
	"github.com/CaioWing/Harbor/internal/storage"
)

type CleanupService struct {
	artRepo    domain.ArtifactRepository
	deployRepo domain.DeploymentRepository
	store      storage.FileStore
	log        *slog.Logger
}

func NewCleanupService(
	artRepo domain.ArtifactRepository,
	deployRepo domain.DeploymentRepository,
	store storage.FileStore,
	log *slog.Logger,
) *CleanupService {
	return &CleanupService{
		artRepo:    artRepo,
		deployRepo: deployRepo,
		store:      store,
		log:        log,
	}
}

// StartScheduler runs cleanup at the specified interval. Call in a goroutine.
func (s *CleanupService) StartScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.log.Info("cleanup scheduler started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("cleanup scheduler stopped")
			return
		case <-ticker.C:
			s.RunCleanup(ctx)
		}
	}
}

// RunCleanup finds artifacts that are not referenced by any active deployment
// and whose storage files no longer match the DB record. It removes orphan
// storage files that have no corresponding artifact in the database.
func (s *CleanupService) RunCleanup(ctx context.Context) {
	s.log.Info("running artifact cleanup")

	artifacts, _, err := s.artRepo.List(ctx, domain.ArtifactFilter{Page: 1, PerPage: 100})
	if err != nil {
		s.log.Warn("cleanup: failed to list artifacts", "err", err)
		return
	}

	cleaned := 0
	for _, art := range artifacts {
		// Check if artifact is referenced by any active deployment
		deployments, _, err := s.deployRepo.List(ctx, domain.DeploymentFilter{
			Page:    1,
			PerPage: 100,
		})
		if err != nil {
			s.log.Warn("cleanup: failed to list deployments", "err", err)
			continue
		}

		inUse := false
		for _, dep := range deployments {
			if dep.ArtifactID == art.ID &&
				(dep.Status == domain.DeploymentStatusScheduled || dep.Status == domain.DeploymentStatusActive) {
				inUse = true
				break
			}
		}

		if inUse {
			continue
		}

		// Verify the storage file exists and matches
		reader, err := s.store.Open(art.StoragePath)
		if err != nil {
			// Storage file is missing — clean up the DB record
			s.log.Info("cleanup: removing artifact with missing storage file",
				"id", art.ID, "name", art.Name, "path", art.StoragePath)
			if err := s.artRepo.Delete(ctx, art.ID); err != nil {
				s.log.Warn("cleanup: failed to delete orphan artifact", "id", art.ID, "err", err)
			} else {
				cleaned++
			}
			continue
		}
		reader.Close()
	}

	s.log.Info("cleanup completed", "removed", cleaned)
}
