package providertest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"vps-billing/internal/provider"
)

// RunProviderContractTests executes the standard provider contract test suite against any Provider implementation.
func RunProviderContractTests(t *testing.T, p provider.Provider) {
	ctx := context.Background()

	t.Run("Health", func(t *testing.T) {
		h, err := p.Health(ctx)
		require.NoError(t, err)
		require.NotNil(t, h)
		assert.Equal(t, "healthy", h.Status)
		assert.NotEmpty(t, h.Version)
	})

	t.Run("Capabilities", func(t *testing.T) {
		caps, err := p.Capabilities(ctx)
		require.NoError(t, err)
		require.NotNil(t, caps)
		assert.True(t, caps.CreateInstance)
		assert.True(t, caps.DeleteInstance)
		assert.NotEmpty(t, caps.SupportedRuntimes)
	})

	t.Run("ListImages", func(t *testing.T) {
		images, err := p.ListImages(ctx, "test-node")
		require.NoError(t, err)
		assert.NotEmpty(t, images)
		assert.NotEmpty(t, images[0].ID)
	})

	t.Run("Lifecycle_Create_Get_Actions_Delete", func(t *testing.T) {
		instID := uuid.New().String()
		opID := uuid.New().String()

		// 1. Create
		createReq := provider.CreateInstanceRequest{
			OperationID:    opID,
			IdempotencyKey: opID,
			NodeID:         "node-1",
			InstanceID:     instID,
			Name:           "test-vm",
			CPUCores:       2,
			MemoryMB:       2048,
			DiskGB:         40,
			Image:          "ubuntu-22.04",
			Virtualization: "kvm",
		}

		op, err := p.CreateInstance(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, op)
		assert.True(t, op.Accepted)

		provInstID := op.Metadata["provider_instance_id"].(string)
		assert.NotEmpty(t, provInstID)

		// 2. Idempotent Create: Same call must succeed without duplicate creation
		op2, err := p.CreateInstance(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, op2)
		assert.Equal(t, provInstID, op2.Metadata["provider_instance_id"])

		// 3. Get Instance
		inst, err := p.GetInstance(ctx, provider.GetInstanceRequest{
			ProviderInstanceID: provInstID,
			PlatformInstanceID: instID,
		})
		require.NoError(t, err)
		assert.Equal(t, provInstID, inst.ProviderInstanceID)
		assert.Equal(t, float64(2), inst.CPUCores)
		assert.Equal(t, int64(2048), inst.MemoryMB)
		assert.NotEmpty(t, inst.IPv4)

		// 4. Stop Instance
		stopOp, err := p.StopInstance(ctx, provider.InstanceActionRequest{
			OperationID:        uuid.New().String(),
			ProviderInstanceID: provInstID,
		})
		require.NoError(t, err)
		assert.True(t, stopOp.Accepted)

		stoppedInst, err := p.GetInstance(ctx, provider.GetInstanceRequest{ProviderInstanceID: provInstID})
		require.NoError(t, err)
		assert.Equal(t, "stopped", stoppedInst.State)

		// 5. Start Instance
		startOp, err := p.StartInstance(ctx, provider.InstanceActionRequest{
			OperationID:        uuid.New().String(),
			ProviderInstanceID: provInstID,
		})
		require.NoError(t, err)
		assert.True(t, startOp.Accepted)

		runningInst, err := p.GetInstance(ctx, provider.GetInstanceRequest{ProviderInstanceID: provInstID})
		require.NoError(t, err)
		assert.Equal(t, "running", runningInst.State)

		// 6. Metrics & Traffic
		usage, err := p.GetUsage(ctx, provider.GetInstanceRequest{ProviderInstanceID: provInstID})
		require.NoError(t, err)
		assert.NotNil(t, usage)

		traffic, err := p.GetTraffic(ctx, provider.GetTrafficRequest{
			GetInstanceRequest: provider.GetInstanceRequest{ProviderInstanceID: provInstID},
			From:               time.Now().Add(-1 * time.Hour),
			To:                 time.Now(),
		})
		require.NoError(t, err)
		assert.NotNil(t, traffic)

		// 7. Port Forwarding
		pfOp, err := p.AddPortForward(ctx, provider.AddPortForwardRequest{
			InstanceActionRequest: provider.InstanceActionRequest{
				OperationID:        uuid.New().String(),
				ProviderInstanceID: provInstID,
			},
			Protocol:   "tcp",
			PublicIP:   "1.2.3.4",
			PublicPort: 10022,
			GuestPort:  22,
		})
		require.NoError(t, err)
		assert.True(t, pfOp.Accepted)

		pfs, err := p.ListPortForwards(ctx, provider.GetInstanceRequest{ProviderInstanceID: provInstID})
		require.NoError(t, err)
		assert.Len(t, pfs, 1)
		assert.Equal(t, 10022, pfs[0].PublicPort)

		delPfOp, err := p.DeletePortForward(ctx, provider.DeletePortForwardRequest{
			InstanceActionRequest: provider.InstanceActionRequest{
				OperationID:        uuid.New().String(),
				ProviderInstanceID: provInstID,
			},
			ProviderMappingID: pfs[0].ProviderMappingID,
		})
		require.NoError(t, err)
		assert.True(t, delPfOp.Accepted)

		pfsAfter, err := p.ListPortForwards(ctx, provider.GetInstanceRequest{ProviderInstanceID: provInstID})
		require.NoError(t, err)
		assert.Empty(t, pfsAfter)

		// 8. Delete Instance
		delOp, err := p.DeleteInstance(ctx, provider.InstanceActionRequest{
			OperationID:        uuid.New().String(),
			ProviderInstanceID: provInstID,
		})
		require.NoError(t, err)
		assert.True(t, delOp.Accepted)

		// 9. Verify Instance is gone
		_, err = p.GetInstance(ctx, provider.GetInstanceRequest{ProviderInstanceID: provInstID})
		require.Error(t, err)
		var provErr *provider.Error
		if errors.As(err, &provErr) {
			assert.Equal(t, provider.ErrCodeInstanceNotFound, provErr.Code)
		}
	})
}
