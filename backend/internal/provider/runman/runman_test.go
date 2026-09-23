package runman_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"vps-billing/internal/provider"
	"vps-billing/internal/provider/providertest"
	"vps-billing/internal/provider/runman"
)

func createConnectedRunmanProvider(t *testing.T, nodeID, token string) (*runman.RunmanProvider, *runman.Gateway) {
	gw := runman.NewGateway(2 * time.Second)
	gw.SetNodeToken(nodeID, token)

	// Register simulated agent that responds successfully to all commands
	_, err := gw.RegisterAgent(nodeID, token, func(cmd runman.AgentCommand) (*runman.AgentCommandResult, error) {
		return &runman.AgentCommandResult{
			CommandID: cmd.CommandID,
			Success:   true,
		}, nil
	})
	require.NoError(t, err)

	prov := runman.NewRunmanProvider(gw)
	return prov, gw
}

func TestRunmanProviderContract(t *testing.T) {
	nodeID := "node-1"
	token := "secret-agent-token"
	prov, _ := createConnectedRunmanProvider(t, nodeID, token)

	// Execute full standardized provider contract test suite
	providertest.RunProviderContractTests(t, prov)
}

func TestRunmanAgentAuth(t *testing.T) {
	gw := runman.NewGateway(5 * time.Second)
	gw.SetNodeToken("node-secure", "correct-token-123")

	// 1. Invalid token should fail
	_, err := gw.RegisterAgent("node-secure", "wrong-token", nil)
	require.Error(t, err)
	var provErr *provider.Error
	require.ErrorAs(t, err, &provErr)
	assert.Equal(t, provider.ErrCodeProviderAuthFailed, provErr.Code)

	// 2. Correct token should succeed
	sess, err := gw.RegisterAgent("node-secure", "correct-token-123", nil)
	require.NoError(t, err)
	assert.Equal(t, "node-secure", sess.NodeID)
}

func TestRunmanHeartbeatAndStateSync(t *testing.T) {
	prov, gw := createConnectedRunmanProvider(t, "node-1", "token-1")
	ctx := context.Background()

	// Create an instance first
	instID := uuid.New().String()
	createRes, err := prov.CreateInstance(ctx, provider.CreateInstanceRequest{
		OperationID:    uuid.New().String(),
		InstanceID:     instID,
		NodeID:         "node-1",
		Name:           "sync-test-vm",
		CPUCores:       2,
		MemoryMB:       2048,
		DiskGB:         40,
		Virtualization: "kvm",
	})
	require.NoError(t, err)
	provInstID := createRes.Metadata["provider_instance_id"].(string)

	// Agent sends heartbeat updating state to "stopped"
	err = gw.HandleHeartbeat("node-1", runman.HeartbeatPayload{
		NodeID:        "node-1",
		Status:        "healthy",
		CPUPercent:    25.0,
		MemoryUsedMB:  1024,
		MemoryTotalMB: 4096,
		DiskUsedGB:    10.0,
		DiskTotalGB:   100.0,
		Instances: map[string]runman.AgentInstanceState{
			provInstID: {
				InstanceID: provInstID,
				State:      "stopped",
				IPv4:       []string{"10.0.0.150"},
			},
		},
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	// GetInstance should reflect state synced from agent heartbeat
	inst, err := prov.GetInstance(ctx, provider.GetInstanceRequest{
		ProviderInstanceID: provInstID,
		NodeID:             "node-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "stopped", inst.State)
	assert.Contains(t, inst.IPv4, "10.0.0.150")
}

func TestRunmanAgentOfflineNeverMisdiagnosedAsDeleted(t *testing.T) {
	gw := runman.NewGateway(100 * time.Millisecond) // short TTL for test
	nodeID := "node-flaky"
	token := "token-flaky"
	gw.SetNodeToken(nodeID, token)

	_, err := gw.RegisterAgent(nodeID, token, func(cmd runman.AgentCommand) (*runman.AgentCommandResult, error) {
		return &runman.AgentCommandResult{CommandID: cmd.CommandID, Success: true}, nil
	})
	require.NoError(t, err)

	prov := runman.NewRunmanProvider(gw)
	ctx := context.Background()

	// Create instance
	instID := uuid.New().String()
	res, err := prov.CreateInstance(ctx, provider.CreateInstanceRequest{
		OperationID:    uuid.New().String(),
		InstanceID:     instID,
		NodeID:         nodeID,
		Name:           "offline-test-vm",
		CPUCores:       1,
		MemoryMB:       1024,
		DiskGB:         20,
		Virtualization: "kvm",
	})
	require.NoError(t, err)
	provInstID := res.Metadata["provider_instance_id"].(string)

	// Wait for agent heartbeat to expire (> 100ms)
	time.Sleep(150 * time.Millisecond)

	// Check instance state: Must NOT be "deleted" or ErrCodeInstanceNotFound!
	// It MUST be returned with state = "unknown"!
	inst, err := prov.GetInstance(ctx, provider.GetInstanceRequest{
		ProviderInstanceID: provInstID,
		NodeID:             nodeID,
	})
	require.NoError(t, err)
	require.NotNil(t, inst)
	assert.Equal(t, "unknown", inst.State, "Offline agent must cause instance state to be 'unknown', NOT deleted!")

	// Trying to execute command on offline node must yield NODE_OFFLINE
	_, err = prov.StopInstance(ctx, provider.InstanceActionRequest{
		ProviderInstanceID: provInstID,
		NodeID:             nodeID,
	})
	require.Error(t, err)
	var provErr *provider.Error
	require.ErrorAs(t, err, &provErr)
	assert.Equal(t, provider.ErrCodeNodeOffline, provErr.Code)

	// Reconnect agent
	_, err = gw.RegisterAgent(nodeID, token, func(cmd runman.AgentCommand) (*runman.AgentCommandResult, error) {
		return &runman.AgentCommandResult{CommandID: cmd.CommandID, Success: true}, nil
	})
	require.NoError(t, err)

	// StopInstance now succeeds!
	stopRes, err := prov.StopInstance(ctx, provider.InstanceActionRequest{
		ProviderInstanceID: provInstID,
		NodeID:             nodeID,
	})
	require.NoError(t, err)
	assert.True(t, stopRes.Accepted)
}
