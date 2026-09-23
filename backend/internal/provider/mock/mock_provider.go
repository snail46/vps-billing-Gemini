package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"vps-billing/internal/provider"
)

type MockProvider struct {
	mu           sync.RWMutex
	name         string
	instances    map[string]*provider.Instance
	portForwards map[string][]provider.PortForward
	nextIP       int
}

func NewMockProvider(name string) *MockProvider {
	if name == "" {
		name = "mock-provider"
	}
	return &MockProvider{
		name:         name,
		instances:    make(map[string]*provider.Instance),
		portForwards: make(map[string][]provider.PortForward),
		nextIP:       10,
	}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Health(ctx context.Context) (*provider.Health, error) {
	return &provider.Health{
		Status:    "healthy",
		Version:   "1.0.0-mock",
		CheckedAt: time.Now().UTC(),
		Details: map[string]any{
			"provider": m.name,
			"driver":   "mock",
		},
	}, nil
}

func (m *MockProvider) Capabilities(ctx context.Context) (*provider.Capabilities, error) {
	return &provider.Capabilities{
		CreateInstance:    true,
		DeleteInstance:    true,
		Start:             true,
		Stop:              true,
		Restart:           true,
		Reinstall:         true,
		ResetPassword:     true,
		Traffic:           true,
		Metrics:           true,
		NAT:               true,
		IPv4:              true,
		IPv6:              true,
		Snapshot:          false,
		Console:           false,
		Firewall:          false,
		SupportedRuntimes: []string{"kvm", "lxc", "docker"},
	}, nil
}

func (m *MockProvider) ListImages(ctx context.Context, nodeID string) ([]provider.Image, error) {
	return []provider.Image{
		{
			ID:          "ubuntu-22.04",
			Name:        "Ubuntu 22.04 LTS",
			OS:          "ubuntu",
			Version:     "22.04",
			Arch:        "x86_64",
			Description: "Ubuntu 22.04 Jammy Jellyfish",
		},
		{
			ID:          "debian-12",
			Name:        "Debian 12 Bookworm",
			OS:          "debian",
			Version:     "12",
			Arch:        "x86_64",
			Description: "Debian 12 Bookworm minimal",
		},
		{
			ID:          "alpine-3.19",
			Name:        "Alpine Linux 3.19",
			OS:          "alpine",
			Version:     "3.19",
			Arch:        "x86_64",
			Description: "Alpine Linux 3.19 container base",
		},
	}, nil
}

func (m *MockProvider) CreateInstance(ctx context.Context, req provider.CreateInstanceRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	provInstID := "mock-vm-" + req.InstanceID

	// Idempotency: if already created with this instance ID, return success immediately
	if inst, exists := m.instances[provInstID]; exists {
		return &provider.Operation{
			ProviderOperationID: "op-" + req.OperationID,
			Status:              "succeeded",
			Accepted:            true,
			Metadata: map[string]any{
				"provider_instance_id": inst.ProviderInstanceID,
				"state":                inst.State,
			},
		}, nil
	}

	m.nextIP++
	ipv4 := fmt.Sprintf("192.168.1.%d", m.nextIP)
	ipv6 := fmt.Sprintf("2001:db8::%d", m.nextIP)

	inst := &provider.Instance{
		ProviderInstanceID: provInstID,
		State:              "running",
		CPUCores:           req.CPUCores,
		MemoryMB:           req.MemoryMB,
		DiskGB:             req.DiskGB,
		IPv4:               []string{ipv4},
		IPv6:               []string{ipv6},
		CreatedAt:          time.Now().UTC(),
		Metadata: map[string]any{
			"image":          req.Image,
			"virtualization": req.Virtualization,
			"node_id":        req.NodeID,
		},
	}

	m.instances[provInstID] = inst

	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
		Metadata: map[string]any{
			"provider_instance_id": provInstID,
			"state":                "running",
		},
	}, nil
}

func (m *MockProvider) GetInstance(ctx context.Context, req provider.GetInstanceRequest) (*provider.Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provInstID := req.ProviderInstanceID
	if provInstID == "" && req.PlatformInstanceID != "" {
		provInstID = "mock-vm-" + req.PlatformInstanceID
	}

	inst, exists := m.instances[provInstID]
	if !exists {
		return nil, provider.NewProviderError(
			provider.ErrCodeInstanceNotFound,
			m.name,
			"404",
			fmt.Sprintf("instance %s not found", provInstID),
			false,
			nil,
		)
	}

	return inst, nil
}

func (m *MockProvider) StartInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.instances[req.ProviderInstanceID]
	if !exists {
		return nil, provider.NewProviderError(
			provider.ErrCodeInstanceNotFound,
			m.name,
			"404",
			fmt.Sprintf("instance %s not found", req.ProviderInstanceID),
			false,
			nil,
		)
	}

	inst.State = "running"
	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (m *MockProvider) StopInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.instances[req.ProviderInstanceID]
	if !exists {
		return nil, provider.NewProviderError(
			provider.ErrCodeInstanceNotFound,
			m.name,
			"404",
			fmt.Sprintf("instance %s not found", req.ProviderInstanceID),
			false,
			nil,
		)
	}

	inst.State = "stopped"
	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (m *MockProvider) RestartInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.instances[req.ProviderInstanceID]
	if !exists {
		return nil, provider.NewProviderError(
			provider.ErrCodeInstanceNotFound,
			m.name,
			"404",
			fmt.Sprintf("instance %s not found", req.ProviderInstanceID),
			false,
			nil,
		)
	}

	inst.State = "running"
	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (m *MockProvider) ReinstallInstance(ctx context.Context, req provider.ReinstallInstanceRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, exists := m.instances[req.ProviderInstanceID]
	if !exists {
		return nil, provider.NewProviderError(
			provider.ErrCodeInstanceNotFound,
			m.name,
			"404",
			fmt.Sprintf("instance %s not found", req.ProviderInstanceID),
			false,
			nil,
		)
	}

	inst.State = "running"
	if req.Image != "" {
		inst.Metadata["image"] = req.Image
	}

	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (m *MockProvider) ResetPassword(ctx context.Context, req provider.ResetPasswordRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.instances[req.ProviderInstanceID]
	if !exists {
		return nil, provider.NewProviderError(
			provider.ErrCodeInstanceNotFound,
			m.name,
			"404",
			fmt.Sprintf("instance %s not found", req.ProviderInstanceID),
			false,
			nil,
		)
	}

	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (m *MockProvider) DeleteInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.instances, req.ProviderInstanceID)
	delete(m.portForwards, req.ProviderInstanceID)

	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (m *MockProvider) GetUsage(ctx context.Context, req provider.GetInstanceRequest) (*provider.Usage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provInstID := req.ProviderInstanceID
	if provInstID == "" && req.PlatformInstanceID != "" {
		provInstID = "mock-vm-" + req.PlatformInstanceID
	}

	inst, exists := m.instances[provInstID]
	if !exists {
		return nil, provider.NewProviderError(
			provider.ErrCodeInstanceNotFound,
			m.name,
			"404",
			fmt.Sprintf("instance %s not found", provInstID),
			false,
			nil,
		)
	}

	return &provider.Usage{
		CPUPercent:    12.5,
		MemoryUsedMB:  inst.MemoryMB / 4,
		MemoryTotalMB: inst.MemoryMB,
		DiskUsedGB:    float64(inst.DiskGB) / 5.0,
		DiskTotalGB:   float64(inst.DiskGB),
		ObservedAt:    time.Now().UTC(),
	}, nil
}

func (m *MockProvider) GetTraffic(ctx context.Context, req provider.GetTrafficRequest) (*provider.Traffic, error) {
	return &provider.Traffic{
		RXBytes: 104857600, // 100 MB
		TXBytes: 52428800,  // 50 MB
		From:    req.From,
		To:      req.To,
	}, nil
}

func (m *MockProvider) ListPortForwards(ctx context.Context, req provider.GetInstanceRequest) ([]provider.PortForward, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pfs := m.portForwards[req.ProviderInstanceID]
	res := make([]provider.PortForward, len(pfs))
	copy(res, pfs)
	return res, nil
}

func (m *MockProvider) AddPortForward(ctx context.Context, req provider.AddPortForwardRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mappingID := fmt.Sprintf("pf-%s-%d", req.Protocol, req.PublicPort)
	pf := provider.PortForward{
		ProviderMappingID: mappingID,
		Protocol:          req.Protocol,
		PublicIP:          req.PublicIP,
		PublicPort:        req.PublicPort,
		GuestPort:         req.GuestPort,
		Description:       req.Description,
	}

	m.portForwards[req.ProviderInstanceID] = append(m.portForwards[req.ProviderInstanceID], pf)

	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
		Metadata: map[string]any{
			"mapping_id": mappingID,
		},
	}, nil
}

func (m *MockProvider) DeletePortForward(ctx context.Context, req provider.DeletePortForwardRequest) (*provider.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := m.portForwards[req.ProviderInstanceID]
	filtered := make([]provider.PortForward, 0, len(list))
	for _, item := range list {
		if item.ProviderMappingID != req.ProviderMappingID {
			filtered = append(filtered, item)
		}
	}
	m.portForwards[req.ProviderInstanceID] = filtered

	return &provider.Operation{
		ProviderOperationID: "op-" + req.OperationID,
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

var _ provider.Provider = (*MockProvider)(nil)
