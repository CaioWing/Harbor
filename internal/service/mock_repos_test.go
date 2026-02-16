package service

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"

	"github.com/CaioWing/Harbor/internal/domain"
)

// --- Mock Device Repository ---

type mockDeviceRepo struct {
	mu      sync.RWMutex
	devices map[uuid.UUID]*domain.Device
	byHash  map[string]*domain.Device
}

func newMockDeviceRepo() *mockDeviceRepo {
	return &mockDeviceRepo{
		devices: make(map[uuid.UUID]*domain.Device),
		byHash:  make(map[string]*domain.Device),
	}
}

func (m *mockDeviceRepo) Create(_ context.Context, d *domain.Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byHash[d.IdentityHash]; exists {
		return domain.ErrConflict
	}
	d.ID = uuid.New()
	m.devices[d.ID] = d
	m.byHash[d.IdentityHash] = d
	return nil
}

func (m *mockDeviceRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if d, ok := m.devices[id]; ok {
		return d, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockDeviceRepo) GetByIdentityHash(_ context.Context, hash string) (*domain.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if d, ok := m.byHash[hash]; ok {
		return d, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockDeviceRepo) List(_ context.Context, f domain.DeviceFilter) ([]*domain.Device, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Device
	for _, d := range m.devices {
		if f.Status != nil && d.Status != *f.Status {
			continue
		}
		if f.DeviceType != nil && d.DeviceType != *f.DeviceType {
			continue
		}
		if len(f.Tags) > 0 && !containsAll(d.Tags, f.Tags) {
			continue
		}
		result = append(result, d)
	}
	return result, len(result), nil
}

func (m *mockDeviceRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.DeviceStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Status = status
	return nil
}

func (m *mockDeviceRepo) UpdateAuthToken(_ context.Context, id uuid.UUID, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.AuthTokenHash = tokenHash
	return nil
}

func (m *mockDeviceRepo) UpdateInventory(_ context.Context, id uuid.UUID, inventory map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Inventory = inventory
	return nil
}

func (m *mockDeviceRepo) UpdateTags(_ context.Context, id uuid.UUID, tags []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Tags = tags
	return nil
}

func (m *mockDeviceRepo) UpdateLastCheckIn(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}

func (m *mockDeviceRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.devices[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Status = domain.DeviceStatusDecommissioned
	return nil
}

func (m *mockDeviceRepo) CountByStatus(_ context.Context) (map[domain.DeviceStatus]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[domain.DeviceStatus]int)
	for _, d := range m.devices {
		counts[d.Status]++
	}
	return counts, nil
}

// --- Mock Artifact Repository ---

type mockArtifactRepo struct {
	mu        sync.RWMutex
	artifacts map[uuid.UUID]*domain.Artifact
}

func newMockArtifactRepo() *mockArtifactRepo {
	return &mockArtifactRepo{
		artifacts: make(map[uuid.UUID]*domain.Artifact),
	}
}

func (m *mockArtifactRepo) Create(_ context.Context, a *domain.Artifact) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.artifacts {
		if existing.Name == a.Name && existing.Version == a.Version {
			return domain.ErrConflict
		}
	}
	a.ID = uuid.New()
	m.artifacts[a.ID] = a
	return nil
}

func (m *mockArtifactRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Artifact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if a, ok := m.artifacts[id]; ok {
		return a, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockArtifactRepo) List(_ context.Context, _ domain.ArtifactFilter) ([]*domain.Artifact, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Artifact
	for _, a := range m.artifacts {
		result = append(result, a)
	}
	return result, len(result), nil
}

func (m *mockArtifactRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.artifacts[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.artifacts, id)
	return nil
}

// --- Mock Deployment Repository ---

type mockDeploymentRepo struct {
	mu          sync.RWMutex
	deployments map[uuid.UUID]*domain.Deployment
	ddEntries   map[uuid.UUID]*domain.DeploymentDevice
	artRepo     *mockArtifactRepo
}

func newMockDeploymentRepo(artRepo *mockArtifactRepo) *mockDeploymentRepo {
	return &mockDeploymentRepo{
		deployments: make(map[uuid.UUID]*domain.Deployment),
		ddEntries:   make(map[uuid.UUID]*domain.DeploymentDevice),
		artRepo:     artRepo,
	}
}

func (m *mockDeploymentRepo) Create(_ context.Context, d *domain.Deployment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d.ID = uuid.New()
	m.deployments[d.ID] = d
	return nil
}

func (m *mockDeploymentRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Deployment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if d, ok := m.deployments[id]; ok {
		return d, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockDeploymentRepo) List(_ context.Context, _ domain.DeploymentFilter) ([]*domain.Deployment, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Deployment
	for _, d := range m.deployments {
		result = append(result, d)
	}
	return result, len(result), nil
}

func (m *mockDeploymentRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.DeploymentStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.deployments[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Status = status
	return nil
}

func (m *mockDeploymentRepo) SetStarted(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.deployments[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Status = domain.DeploymentStatusActive
	return nil
}

func (m *mockDeploymentRepo) SetFinished(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.deployments[id]
	if !ok {
		return domain.ErrNotFound
	}
	d.Status = domain.DeploymentStatusCompleted
	return nil
}

func (m *mockDeploymentRepo) GetStats(_ context.Context) (*domain.DeploymentStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := &domain.DeploymentStats{}
	for _, d := range m.deployments {
		stats.Total++
		switch d.Status {
		case domain.DeploymentStatusScheduled:
			stats.Scheduled++
		case domain.DeploymentStatusActive:
			stats.Active++
		case domain.DeploymentStatusCompleted:
			stats.Completed++
		case domain.DeploymentStatusCancelled:
			stats.Cancelled++
		}
	}
	return stats, nil
}

func (m *mockDeploymentRepo) CreateDeploymentDevice(_ context.Context, dd *domain.DeploymentDevice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.ddEntries {
		if existing.DeploymentID == dd.DeploymentID && existing.DeviceID == dd.DeviceID {
			return domain.ErrConflict
		}
	}
	dd.ID = uuid.New()
	m.ddEntries[dd.ID] = dd
	return nil
}

func (m *mockDeploymentRepo) GetDeploymentDevices(_ context.Context, deploymentID uuid.UUID) ([]*domain.DeploymentDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.DeploymentDevice
	for _, dd := range m.ddEntries {
		if dd.DeploymentID == deploymentID {
			result = append(result, dd)
		}
	}
	return result, nil
}

func (m *mockDeploymentRepo) GetPendingDeploymentForDevice(_ context.Context, deviceID uuid.UUID) (*domain.DeploymentDevice, *domain.Deployment, *domain.Artifact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, dd := range m.ddEntries {
		if dd.DeviceID == deviceID && dd.Status == domain.DDStatusPending {
			dep := m.deployments[dd.DeploymentID]
			if dep != nil && dep.Status == domain.DeploymentStatusActive {
				art, _ := m.artRepo.GetByID(context.Background(), dep.ArtifactID)
				return dd, dep, art, nil
			}
		}
	}
	return nil, nil, nil, domain.ErrNotFound
}

func (m *mockDeploymentRepo) UpdateDeploymentDeviceStatus(_ context.Context, id uuid.UUID, status domain.DeploymentDeviceStatus, log string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	dd, ok := m.ddEntries[id]
	if !ok {
		return domain.ErrNotFound
	}
	dd.Status = status
	dd.Log = log
	return nil
}

func (m *mockDeploymentRepo) CountDeploymentDevicesByStatus(_ context.Context, deploymentID uuid.UUID) (map[domain.DeploymentDeviceStatus]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[domain.DeploymentDeviceStatus]int)
	for _, dd := range m.ddEntries {
		if dd.DeploymentID == deploymentID {
			counts[dd.Status]++
		}
	}
	return counts, nil
}

// --- Mock File Store ---

type mockFileStore struct {
	mu    sync.RWMutex
	files map[string][]byte
}

func newMockFileStore() *mockFileStore {
	return &mockFileStore{files: make(map[string][]byte)}
}

func (m *mockFileStore) Save(name string, reader io.Reader) (string, int64, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", 0, err
	}
	path := "/mock/storage/" + name
	m.mu.Lock()
	m.files[path] = data
	m.mu.Unlock()
	return path, int64(len(data)), nil
}

func (m *mockFileStore) Open(path string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.files[path]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return io.NopCloser(io.NewSectionReader(newBytesReaderAt(data), 0, int64(len(data)))), nil
}

func (m *mockFileStore) Delete(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, path)
	return nil
}

// Helper: bytes.Reader implements io.ReaderAt
type bytesReaderAt struct {
	data []byte
}

func newBytesReaderAt(data []byte) *bytesReaderAt {
	return &bytesReaderAt{data: data}
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Helper function
func containsAll(haystack []string, needles []string) bool {
	set := make(map[string]bool, len(haystack))
	for _, s := range haystack {
		set[s] = true
	}
	for _, n := range needles {
		if !set[n] {
			return false
		}
	}
	return true
}
