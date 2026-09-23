package runman

import (
	"encoding/json"
	"time"
)

type AgentInstanceState struct {
	InstanceID string   `json:"instance_id"`
	State      string   `json:"state"` // "running", "stopped", "provisioning", "error"
	CPUCores   float64  `json:"cpu_cores"`
	MemoryMB   int64    `json:"memory_mb"`
	DiskGB     int64    `json:"disk_gb"`
	IPv4       []string `json:"ipv4"`
	IPv6       []string `json:"ipv6"`
	RXBytes    int64    `json:"rx_bytes"`
	TXBytes    int64    `json:"tx_bytes"`
}

type HeartbeatPayload struct {
	NodeID        string                        `json:"node_id"`
	Status        string                        `json:"status"` // "healthy", "degraded"
	CPUPercent    float64                       `json:"cpu_percent"`
	MemoryUsedMB  int64                         `json:"memory_used_mb"`
	MemoryTotalMB int64                         `json:"memory_total_mb"`
	DiskUsedGB    float64                       `json:"disk_used_gb"`
	DiskTotalGB   float64                       `json:"disk_total_gb"`
	Instances     map[string]AgentInstanceState `json:"instances"`
	Timestamp     time.Time                     `json:"timestamp"`
}

type AgentCommand struct {
	CommandID string          `json:"command_id"`
	NodeID    string          `json:"node_id"`
	Action    string          `json:"action"` // "create", "start", "stop", "restart", "reinstall", "reset_password", "delete", "add_port_forward", "delete_port_forward"
	Payload   json.RawMessage `json:"payload"`
	TimeoutMs int             `json:"timeout_ms"`
}

type AgentCommandResult struct {
	CommandID    string          `json:"command_id"`
	Success      bool            `json:"success"`
	ErrorCode    string          `json:"error_code,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	Output       json.RawMessage `json:"output,omitempty"`
}

type PortForwardRule struct {
	MappingID   string `json:"mapping_id"`
	Protocol    string `json:"protocol"`
	PublicIP    string `json:"public_ip"`
	PublicPort  int    `json:"public_port"`
	GuestPort   int    `json:"guest_port"`
	Description string `json:"description"`
}
