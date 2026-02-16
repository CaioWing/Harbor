package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/domain"
)

func newTestArtifactService() (*ArtifactService, *mockArtifactRepo, *mockFileStore) {
	repo := newMockArtifactRepo()
	store := newMockFileStore()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewArtifactService(repo, store, log)
	return svc, repo, store
}

func TestArtifactCreate_Success(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		Description: "test artifact",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("binary content here"),
	}

	artifact, err := svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artifact.Name != "myapp" {
		t.Fatalf("expected name myapp, got %s", artifact.Name)
	}
	if artifact.Version != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", artifact.Version)
	}
	if artifact.ChecksumSHA256 == "" {
		t.Fatal("expected non-empty checksum")
	}
	if artifact.FileSize != 19 { // len("binary content here")
		t.Fatalf("expected file size 19, got %d", artifact.FileSize)
	}
	if artifact.FileMode != "0644" {
		t.Fatalf("expected default file mode 0644, got %s", artifact.FileMode)
	}
}

func TestArtifactCreate_CustomFileMode(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		FileMode:    "0755",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("content"),
	}

	artifact, err := svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artifact.FileMode != "0755" {
		t.Fatalf("expected file mode 0755, got %s", artifact.FileMode)
	}
}

func TestArtifactCreate_MissingName(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Version:     "1.0.0",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("content"),
	}

	_, err := svc.Create(ctx, input)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestArtifactCreate_MissingVersion(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("content"),
	}

	_, err := svc.Create(ctx, input)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestArtifactCreate_MissingTargetPath(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		FileName:    "myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("content"),
	}

	_, err := svc.Create(ctx, input)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestArtifactCreate_NoDeviceTypes(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{},
		File:        strings.NewReader("content"),
	}

	_, err := svc.Create(ctx, input)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestArtifactCreate_DuplicateNameVersion(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("content"),
	}

	_, err := svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}

	input.File = strings.NewReader("content2")
	_, err = svc.Create(ctx, input)
	if err == nil {
		t.Fatal("expected error on duplicate create")
	}
}

func TestArtifactGetByID(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("content"),
	}
	created, _ := svc.Create(ctx, input)

	found, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Name != "myapp" {
		t.Fatalf("expected myapp, got %s", found.Name)
	}
}

func TestArtifactGetByID_NotFound(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	_, err := svc.GetByID(ctx, uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestArtifactDelete(t *testing.T) {
	svc, repo, store := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("content"),
	}
	created, _ := svc.Create(ctx, input)

	err := svc.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Artifact should be gone from repo
	if len(repo.artifacts) != 0 {
		t.Fatalf("expected 0 artifacts, got %d", len(repo.artifacts))
	}

	// File should be gone from store
	if len(store.files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(store.files))
	}
}

func TestArtifactDelete_NotFound(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	err := svc.Delete(ctx, uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestArtifactList(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		svc.Create(ctx, CreateArtifactInput{
			Name:        "myapp",
			Version:     string(rune('1'+i)) + ".0.0",
			FileName:    "myapp",
			TargetPath:  "/usr/local/bin/myapp",
			DeviceTypes: []string{"raspberry-pi-4"},
			File:        strings.NewReader("content"),
		})
	}

	artifacts, total, err := svc.List(ctx, domain.ArtifactFilter{Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(artifacts) != 3 {
		t.Fatalf("expected 3 artifacts, got %d", len(artifacts))
	}
}

func TestArtifactOpenFile(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:        "myapp",
		Version:     "1.0.0",
		FileName:    "myapp",
		TargetPath:  "/usr/local/bin/myapp",
		DeviceTypes: []string{"raspberry-pi-4"},
		File:        strings.NewReader("binary data"),
	}
	created, _ := svc.Create(ctx, input)

	reader, artifact, err := svc.OpenFile(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	if artifact.Name != "myapp" {
		t.Fatalf("expected myapp, got %s", artifact.Name)
	}
}

func TestArtifactCreate_WithHooks(t *testing.T) {
	svc, _, _ := newTestArtifactService()
	ctx := context.Background()

	input := CreateArtifactInput{
		Name:           "myapp",
		Version:        "1.0.0",
		FileName:       "myapp",
		TargetPath:     "/usr/local/bin/myapp",
		DeviceTypes:    []string{"raspberry-pi-4"},
		PreInstallCmd:  "systemctl stop myapp",
		PostInstallCmd: "systemctl start myapp",
		RollbackCmd:    "systemctl restart myapp-old",
		File:           strings.NewReader("content"),
	}

	artifact, err := svc.Create(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artifact.PreInstallCmd != "systemctl stop myapp" {
		t.Fatal("pre_install_cmd not set")
	}
	if artifact.PostInstallCmd != "systemctl start myapp" {
		t.Fatal("post_install_cmd not set")
	}
	if artifact.RollbackCmd != "systemctl restart myapp-old" {
		t.Fatal("rollback_cmd not set")
	}
}
