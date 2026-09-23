package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Provider struct {
	ID                uuid.UUID       `json:"id"`
	Name              string          `json:"name"`
	ProviderType      string          `json:"provider_type"`
	Endpoint          *string         `json:"endpoint,omitempty"`
	CredentialRef     *string         `json:"credential_ref,omitempty"`
	Status            string          `json:"status"`
	Version           *string         `json:"version,omitempty"`
	Config            json.RawMessage `json:"config"`
	Capabilities      json.RawMessage `json:"capabilities"`
	LastHealthCheckAt *time.Time      `json:"last_health_check_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type NodeGroup struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Node struct {
	ID                uuid.UUID       `json:"id"`
	ProviderID        uuid.UUID       `json:"provider_id"`
	NodeGroupID       *uuid.UUID      `json:"node_group_id,omitempty"`
	ProviderNodeID    *string         `json:"provider_node_id,omitempty"`
	Name              string          `json:"name"`
	Region            string          `json:"region"`
	Status            string          `json:"status"`
	CPUTotal          float64         `json:"cpu_total"`
	MemoryTotalMB     int64           `json:"memory_total_mb"`
	DiskTotalGB       int64           `json:"disk_total_gb"`
	CPUAllocated      float64         `json:"cpu_allocated"`
	MemoryAllocatedMB int64           `json:"memory_allocated_mb"`
	DiskAllocatedGB   int64           `json:"disk_allocated_gb"`
	CPUReserved       float64         `json:"cpu_reserved"`
	MemoryReservedMB  int64           `json:"memory_reserved_mb"`
	DiskReservedGB    int64           `json:"disk_reserved_gb"`
	Weight            int             `json:"weight"`
	Capabilities      json.RawMessage `json:"capabilities"`
	LastSeenAt        *time.Time      `json:"last_seen_at,omitempty"`
	Version           int64           `json:"version"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// AvailableCPU returns unreserved and unallocated CPU cores.
func (n *Node) AvailableCPU() float64 {
	avail := n.CPUTotal - (n.CPUAllocated + n.CPUReserved)
	if avail < 0 {
		return 0
	}
	return avail
}

// AvailableMemoryMB returns unreserved and unallocated Memory in MB.
func (n *Node) AvailableMemoryMB() int64 {
	avail := n.MemoryTotalMB - (n.MemoryAllocatedMB + n.MemoryReservedMB)
	if avail < 0 {
		return 0
	}
	return avail
}

// AvailableDiskGB returns unreserved and unallocated Disk in GB.
func (n *Node) AvailableDiskGB() int64 {
	avail := n.DiskTotalGB - (n.DiskAllocatedGB + n.DiskReservedGB)
	if avail < 0 {
		return 0
	}
	return avail
}

type ResourceReservation struct {
	ID           uuid.UUID `json:"id"`
	NodeID       uuid.UUID `json:"node_id"`
	OperationID  uuid.UUID `json:"operation_id"`
	CPUCores     float64   `json:"cpu_cores"`
	MemoryMB     int64     `json:"memory_mb"`
	DiskGB       int64     `json:"disk_gb"`
	IPv4Count    int       `json:"ipv4_count"`
	IPv6Count    int       `json:"ipv6_count"`
	NATPortCount int       `json:"nat_port_count"`
	Status       string    `json:"status"` // reserved, committed, released, expired
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Instance struct {
	ID                 uuid.UUID  `json:"id"`
	SubscriptionID     uuid.UUID  `json:"subscription_id"`
	NodeID             *uuid.UUID `json:"node_id,omitempty"`
	ProviderID         *uuid.UUID `json:"provider_id,omitempty"`
	ProviderInstanceID *string    `json:"provider_instance_id,omitempty"`
	Name               string     `json:"name"`
	DesiredState       string     `json:"desired_state"`
	ObservedState      string     `json:"observed_state"`
	CPUCores           float64    `json:"cpu_cores"`
	MemoryMB           int        `json:"memory_mb"`
	DiskGB             int        `json:"disk_gb"`
	TrafficLimitGB     *int64     `json:"traffic_limit_gb,omitempty"`
	BandwidthMbps      *int       `json:"bandwidth_mbps,omitempty"`
	ImageID            *string    `json:"image_id,omitempty"`
	PrimaryIPv4        *string    `json:"primary_ipv4,omitempty"`
	PrimaryIPv6        *string    `json:"primary_ipv6,omitempty"`
	LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`
	Version            int64      `json:"version"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}
