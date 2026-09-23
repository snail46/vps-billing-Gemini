package lxdapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"vps-billing/internal/provider"
	"vps-billing/internal/provider/lxdapi"
)

// Fake LXD server simulating the LXD REST API
func newFakeLXDServer() (*httptest.Server, *sync.Map) {
	instances := &sync.Map{}

	mux := http.NewServeMux()

	// GET /1.0
	mux.HandleFunc("/1.0", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]any{
			"type":        "sync",
			"status":      "Success",
			"status_code": 200,
			"metadata": map[string]any{
				"environment": map[string]any{
					"server_version": "5.21",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// GET /1.0/images
	mux.HandleFunc("/1.0/images", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"type":        "sync",
			"status":      "Success",
			"status_code": 200,
			"metadata":    []any{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	// POST /1.0/instances
	mux.HandleFunc("/1.0/instances", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			name := body["name"].(string)

			if _, exists := instances.Load(name); exists {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type":        "error",
					"error":       "instance already exists",
					"error_code":  409,
					"status_code": 409,
				})
				return
			}

			instances.Store(name, "running")
			resp := map[string]any{
				"type":        "async",
				"status":      "Operation created",
				"status_code": 100,
				"operation":   "/1.0/operations/op-123",
				"metadata": map[string]any{
					"id": "/1.0/operations/op-123",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	})

	// /1.0/instances/<name>
	mux.HandleFunc("/1.0/instances/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/1.0/instances/"):]
		if idx := len(name) - len("/state"); idx > 0 && name[idx:] == "/state" {
			instName := name[:idx]
			if _, ok := instances.Load(instName); !ok {
				http.NotFound(w, r)
				return
			}
			if r.Method == "PUT" {
				var action map[string]string
				_ = json.NewDecoder(r.Body).Decode(&action)
				if action["action"] == "stop" {
					instances.Store(instName, "stopped")
				} else if action["action"] == "start" {
					instances.Store(instName, "running")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type":        "sync",
					"status":      "Success",
					"status_code": 200,
				})
				return
			}
			return
		}

		if r.Method == "GET" {
			state, ok := instances.Load(name)
			if !ok {
				http.NotFound(w, r)
				return
			}
			resp := map[string]any{
				"type":        "sync",
				"status":      "Success",
				"status_code": 200,
				"metadata": map[string]any{
					"name":   name,
					"status": state.(string),
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == "DELETE" {
			instances.Delete(name)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":        "sync",
				"status":      "Success",
				"status_code": 200,
			})
			return
		}
	})

	server := httptest.NewServer(mux)
	return server, instances
}

func TestLXDAPIAdapterContract(t *testing.T) {
	ctx := context.Background()
	fakeServer, _ := newFakeLXDServer()
	defer fakeServer.Close()

	adapter := lxdapi.NewAdapter("test-lxd", lxdapi.Config{
		Endpoint: fakeServer.URL,
		Token:    "test-token",
	}, fakeServer.Client())

	// 1. Health
	t.Run("Health", func(t *testing.T) {
		h, err := adapter.Health(ctx)
		require.NoError(t, err)
		assert.Equal(t, "healthy", h.Status)
		assert.Equal(t, "5.21", h.Version)
	})

	// 2. Capabilities
	t.Run("Capabilities", func(t *testing.T) {
		caps, err := adapter.Capabilities(ctx)
		require.NoError(t, err)
		assert.True(t, caps.CreateInstance)
		assert.False(t, caps.ResetPassword) // Returns unsupported for password reset
	})

	// 3. Lifecycle: Create -> Get -> Stop -> Start -> Delete
	t.Run("Lifecycle_Create_Get_Actions_Delete", func(t *testing.T) {
		instID := uuid.New().String()
		opID := uuid.New().String()

		createReq := provider.CreateInstanceRequest{
			OperationID:    opID,
			IdempotencyKey: opID,
			NodeID:         "node-1",
			InstanceID:     instID,
			Name:           "vm-" + instID[:8],
			CPUCores:       2,
			MemoryMB:       2048,
			Image:          "ubuntu/22.04",
			Virtualization: "lxc",
		}

		// Create
		op, err := adapter.CreateInstance(ctx, createReq)
		require.NoError(t, err)
		assert.True(t, op.Accepted)

		// Idempotency: Create duplicate succeeds
		op2, err := adapter.CreateInstance(ctx, createReq)
		require.NoError(t, err)
		assert.True(t, op2.Accepted)

		// Get
		inst, err := adapter.GetInstance(ctx, provider.GetInstanceRequest{
			ProviderInstanceID: createReq.Name,
		})
		require.NoError(t, err)
		assert.Equal(t, createReq.Name, inst.ProviderInstanceID)
		assert.Equal(t, "running", inst.State)

		// Stop
		stopOp, err := adapter.StopInstance(ctx, provider.InstanceActionRequest{
			OperationID:        uuid.New().String(),
			ProviderInstanceID: createReq.Name,
		})
		require.NoError(t, err)
		assert.True(t, stopOp.Accepted)

		stoppedInst, err := adapter.GetInstance(ctx, provider.GetInstanceRequest{
			ProviderInstanceID: createReq.Name,
		})
		require.NoError(t, err)
		assert.Equal(t, "stopped", stoppedInst.State)

		// Unsupported Operation: Reset Password
		_, err = adapter.ResetPassword(ctx, provider.ResetPasswordRequest{
			InstanceActionRequest: provider.InstanceActionRequest{
				ProviderInstanceID: createReq.Name,
			},
			RootPassword: "newpassword",
		})
		require.Error(t, err)
		var provErr *provider.Error
		require.True(t, errors.As(err, &provErr))
		assert.Equal(t, provider.ErrCodeUnsupportedOperation, provErr.Code)

		// Delete
		delOp, err := adapter.DeleteInstance(ctx, provider.InstanceActionRequest{
			OperationID:        uuid.New().String(),
			ProviderInstanceID: createReq.Name,
		})
		require.NoError(t, err)
		assert.True(t, delOp.Accepted)

		// Verify Deleted
		_, err = adapter.GetInstance(ctx, provider.GetInstanceRequest{
			ProviderInstanceID: createReq.Name,
		})
		require.Error(t, err)
		require.True(t, errors.As(err, &provErr))
		assert.Equal(t, provider.ErrCodeInstanceNotFound, provErr.Code)
	})
}
