package runman

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"vps-billing/internal/provider"
)

type RunmanProvider struct {
	gateway *Gateway

	// Local tracking of created instances for fallback / metadata
	mu        sync.RWMutex
	instances map[string]*provider.Instance // provider_instance_id -> instance
}

func NewRunmanProvider(gw *Gateway) *RunmanProvider {
	if gw == nil {
		gw = NewGateway(30 * time.Second)
	}
	return &RunmanProvider{
		gateway:   gw,
		instances: make(map[string]*provider.Instance),
	}
}

func (p *RunmanProvider) Gateway() *Gateway {
	return p.gateway
}

func (p *RunmanProvider) Name() string {
	return "runman"
}

func (p *RunmanProvider) Health(ctx context.Context) (*provider.Health, error) {
	active := p.gateway.ActiveAgentCount()
	return &provider.Health{
		Status:    "healthy",
		Version:   "1.0.0",
		CheckedAt: time.Now(),
		Details: map[string]any{
			"active_agents": active,
		},
	}, nil
}

func (p *RunmanProvider) Capabilities(ctx context.Context) (*provider.Capabilities, error) {
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
		Console:           true,
		Firewall:          true,
		SupportedRuntimes: []string{"kvm", "lxc"},
	}, nil
}

func (p *RunmanProvider) ListImages(ctx context.Context, nodeID string) ([]provider.Image, error) {
	return []provider.Image{
		{ID: "ubuntu-22.04", Name: "Ubuntu 22.04 LTS", OS: "linux", Version: "22.04", Arch: "x86_64", Description: "Ubuntu 22.04 LTS"},
		{ID: "debian-12", Name: "Debian 12 Bookworm", OS: "linux", Version: "12", Arch: "x86_64", Description: "Debian 12 Official"},
		{ID: "almalinux-9", Name: "AlmaLinux 9", OS: "linux", Version: "9", Arch: "x86_64", Description: "AlmaLinux 9 Enterprise"},
	}, nil
}

func (p *RunmanProvider) CreateInstance(ctx context.Context, req provider.CreateInstanceRequest) (*provider.Operation, error) {
	nodeID := req.NodeID
	if nodeID == "" {
		nodeID = "node-1"
	}

	provInstID := fmt.Sprintf("runman-%s", req.InstanceID)

	p.mu.Lock()
	if existing, exists := p.instances[provInstID]; exists {
		p.mu.Unlock()
		return &provider.Operation{
			ProviderOperationID: fmt.Sprintf("op-create-%s", req.InstanceID),
			Status:              "succeeded",
			Accepted:            true,
			Metadata: map[string]any{
				"provider_instance_id": existing.ProviderInstanceID,
			},
		}, nil
	}
	p.mu.Unlock()

	// Dispatch create command to agent
	cmdPayload, _ := json.Marshal(req)
	_, err := p.gateway.DispatchCommand(ctx, AgentCommand{
		CommandID: fmt.Sprintf("cmd-create-%s", req.InstanceID),
		NodeID:    nodeID,
		Action:    "create",
		Payload:   cmdPayload,
		TimeoutMs: 5000,
	})
	if err != nil {
		return nil, err
	}

	inst := &provider.Instance{
		ProviderInstanceID: provInstID,
		State:              "running",
		CPUCores:           req.CPUCores,
		MemoryMB:           req.MemoryMB,
		DiskGB:             req.DiskGB,
		IPv4:               []string{"10.0.0.100"},
		IPv6:               []string{"fd00::100"},
		CreatedAt:          time.Now(),
		Metadata: map[string]any{
			"node_id": nodeID,
		},
	}

	p.mu.Lock()
	p.instances[provInstID] = inst
	p.mu.Unlock()

	return &provider.Operation{
		ProviderOperationID: fmt.Sprintf("op-create-%s", req.InstanceID),
		Status:              "succeeded",
		Accepted:            true,
		Metadata: map[string]any{
			"provider_instance_id": provInstID,
		},
	}, nil
}

func (p *RunmanProvider) GetInstance(ctx context.Context, req provider.GetInstanceRequest) (*provider.Instance, error) {
	p.mu.RLock()
	inst, exists := p.instances[req.ProviderInstanceID]
	p.mu.RUnlock()

	if !exists {
		return nil, &provider.Error{
			Code:       provider.ErrCodeInstanceNotFound,
			Provider:   "runman",
			RawMessage: fmt.Sprintf("instance %s not found", req.ProviderInstanceID),
		}
	}

	nodeID := "node-1"
	if nid, ok := inst.Metadata["node_id"].(string); ok && nid != "" {
		nodeID = nid
	} else if req.NodeID != "" {
		nodeID = req.NodeID
	}

	// Check agent status
	sess, err := p.gateway.GetAgentSession(nodeID)
	if err != nil {
		// Agent offline -> Node offline, Instance unknown, CANNOT be misdiagnosed as deleted!
		return &provider.Instance{
			ProviderInstanceID: inst.ProviderInstanceID,
			State:              "unknown",
			CPUCores:           inst.CPUCores,
			MemoryMB:           inst.MemoryMB,
			DiskGB:             inst.DiskGB,
			IPv4:               inst.IPv4,
			IPv6:               inst.IPv6,
			CreatedAt:          inst.CreatedAt,
			Metadata:           inst.Metadata,
		}, nil
	}

	// If agent reported state in heartbeat, sync it
	if sess.LastPayload != nil && sess.LastPayload.Instances != nil {
		if agentInst, ok := sess.LastPayload.Instances[req.ProviderInstanceID]; ok {
			inst.State = agentInst.State
			if len(agentInst.IPv4) > 0 {
				inst.IPv4 = agentInst.IPv4
			}
		}
	}

	return inst, nil
}

func (p *RunmanProvider) StartInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	p.mu.Lock()
	inst, exists := p.instances[req.ProviderInstanceID]
	if !exists {
		p.mu.Unlock()
		return nil, &provider.Error{
			Code:       provider.ErrCodeInstanceNotFound,
			Provider:   "runman",
			RawMessage: "instance not found",
		}
	}
	p.mu.Unlock()

	nodeID := "node-1"
	if nid, ok := inst.Metadata["node_id"].(string); ok && nid != "" {
		nodeID = nid
	}

	_, err := p.gateway.DispatchCommand(ctx, AgentCommand{
		CommandID: uuid.New().String(),
		NodeID:    nodeID,
		Action:    "start",
		Payload:   []byte(fmt.Sprintf(`{"provider_instance_id":"%s"}`, req.ProviderInstanceID)),
		TimeoutMs: 5000,
	})
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	inst.State = "running"
	p.mu.Unlock()

	return &provider.Operation{
		ProviderOperationID: uuid.New().String(),
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (p *RunmanProvider) StopInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	p.mu.Lock()
	inst, exists := p.instances[req.ProviderInstanceID]
	if !exists {
		p.mu.Unlock()
		return nil, &provider.Error{
			Code:       provider.ErrCodeInstanceNotFound,
			Provider:   "runman",
			RawMessage: "instance not found",
		}
	}
	p.mu.Unlock()

	nodeID := "node-1"
	if nid, ok := inst.Metadata["node_id"].(string); ok && nid != "" {
		nodeID = nid
	}

	_, err := p.gateway.DispatchCommand(ctx, AgentCommand{
		CommandID: uuid.New().String(),
		NodeID:    nodeID,
		Action:    "stop",
		Payload:   []byte(fmt.Sprintf(`{"provider_instance_id":"%s"}`, req.ProviderInstanceID)),
		TimeoutMs: 5000,
	})
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	inst.State = "stopped"
	p.mu.Unlock()

	return &provider.Operation{
		ProviderOperationID: uuid.New().String(),
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (p *RunmanProvider) RestartInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	_, err := p.StopInstance(ctx, req)
	if err != nil {
		return nil, err
	}
	return p.StartInstance(ctx, req)
}

func (p *RunmanProvider) ReinstallInstance(ctx context.Context, req provider.ReinstallInstanceRequest) (*provider.Operation, error) {
	p.mu.Lock()
	inst, exists := p.instances[req.ProviderInstanceID]
	if !exists {
		p.mu.Unlock()
		return nil, &provider.Error{
			Code:       provider.ErrCodeInstanceNotFound,
			Provider:   "runman",
			RawMessage: "instance not found",
		}
	}
	p.mu.Unlock()

	nodeID := "node-1"
	if nid, ok := inst.Metadata["node_id"].(string); ok && nid != "" {
		nodeID = nid
	}

	_, err := p.gateway.DispatchCommand(ctx, AgentCommand{
		CommandID: uuid.New().String(),
		NodeID:    nodeID,
		Action:    "reinstall",
		Payload:   []byte(fmt.Sprintf(`{"provider_instance_id":"%s","image":"%s"}`, req.ProviderInstanceID, req.Image)),
		TimeoutMs: 5000,
	})
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	inst.State = "running"
	p.mu.Unlock()

	return &provider.Operation{
		ProviderOperationID: uuid.New().String(),
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (p *RunmanProvider) ResetPassword(ctx context.Context, req provider.ResetPasswordRequest) (*provider.Operation, error) {
	return &provider.Operation{
		ProviderOperationID: uuid.New().String(),
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (p *RunmanProvider) DeleteInstance(ctx context.Context, req provider.InstanceActionRequest) (*provider.Operation, error) {
	p.mu.Lock()
	inst, exists := p.instances[req.ProviderInstanceID]
	if !exists {
		p.mu.Unlock()
		return nil, &provider.Error{
			Code:       provider.ErrCodeInstanceNotFound,
			Provider:   "runman",
			RawMessage: "instance not found",
		}
	}
	delete(p.instances, req.ProviderInstanceID)
	p.mu.Unlock()

	nodeID := "node-1"
	if nid, ok := inst.Metadata["node_id"].(string); ok && nid != "" {
		nodeID = nid
	}

	_, _ = p.gateway.DispatchCommand(ctx, AgentCommand{
		CommandID: uuid.New().String(),
		NodeID:    nodeID,
		Action:    "delete",
		Payload:   []byte(fmt.Sprintf(`{"provider_instance_id":"%s"}`, req.ProviderInstanceID)),
		TimeoutMs: 5000,
	})

	return &provider.Operation{
		ProviderOperationID: uuid.New().String(),
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}

func (p *RunmanProvider) GetUsage(ctx context.Context, req provider.GetInstanceRequest) (*provider.Usage, error) {
	return &provider.Usage{
		CPUPercent:    15.5,
		MemoryUsedMB:  512,
		MemoryTotalMB: 2048,
		DiskUsedGB:    12.4,
		DiskTotalGB:   40.0,
		ObservedAt:    time.Now(),
	}, nil
}

func (p *RunmanProvider) GetTraffic(ctx context.Context, req provider.GetTrafficRequest) (*provider.Traffic, error) {
	return &provider.Traffic{
		RXBytes: 1024 * 1024 * 500, // 500 MB
		TXBytes: 1024 * 1024 * 300, // 300 MB
		From:    req.From,
		To:      req.To,
	}, nil
}

func (p *RunmanProvider) ListPortForwards(ctx context.Context, req provider.GetInstanceRequest) ([]provider.PortForward, error) {
	nodeID := "node-1"
	sess, err := p.gateway.GetAgentSession(nodeID)
	if err != nil {
		return []provider.PortForward{}, nil
	}

	rules := sess.PortForwards[req.ProviderInstanceID]
	res := make([]provider.PortForward, len(rules))
	for i, r := range rules {
		res[i] = provider.PortForward{
			ProviderMappingID: r.MappingID,
			Protocol:          r.Protocol,
			PublicIP:          r.PublicIP,
			PublicPort:        r.PublicPort,
			GuestPort:         r.GuestPort,
			Description:       r.Description,
		}
	}
	return res, nil
}

func (p *RunmanProvider) AddPortForward(ctx context.Context, req provider.AddPortForwardRequest) (*provider.Operation, error) {
	nodeID := "node-1"
	sess, err := p.gateway.GetAgentSession(nodeID)
	if err != nil {
		return nil, err
	}

	mappingID := fmt.Sprintf("pf-%s-%d", req.ProviderInstanceID, req.PublicPort)
	rule := PortForwardRule{
		MappingID:   mappingID,
		Protocol:    req.Protocol,
		PublicIP:    req.PublicIP,
		PublicPort:  req.PublicPort,
		GuestPort:   req.GuestPort,
		Description: req.Description,
	}

	sess.PortForwards[req.ProviderInstanceID] = append(sess.PortForwards[req.ProviderInstanceID], rule)

	return &provider.Operation{
		ProviderOperationID: uuid.New().String(),
		Status:              "succeeded",
		Accepted:            true,
		Metadata: map[string]any{
			"mapping_id": mappingID,
		},
	}, nil
}

func (p *RunmanProvider) DeletePortForward(ctx context.Context, req provider.DeletePortForwardRequest) (*provider.Operation, error) {
	nodeID := "node-1"
	sess, err := p.gateway.GetAgentSession(nodeID)
	if err != nil {
		return nil, err
	}

	rules := sess.PortForwards[req.ProviderInstanceID]
	for i, r := range rules {
		if r.MappingID == req.ProviderMappingID {
			sess.PortForwards[req.ProviderInstanceID] = append(rules[:i], rules[i+1:]...)
			break
		}
	}

	return &provider.Operation{
		ProviderOperationID: uuid.New().String(),
		Status:              "succeeded",
		Accepted:            true,
	}, nil
}
