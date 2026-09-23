package scheduler

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	domainInfrastructure "vps-billing/internal/domain/infrastructure"
)

type ScheduleRequirements struct {
	OperationID  uuid.UUID
	NodeGroupID  *uuid.UUID
	CPUCores     float64
	MemoryMB     int64
	DiskGB       int64
	IPv4Count    int
	IPv6Count    int
	NATPortCount int
	TTL          time.Duration
}

type Scheduler struct {
	infraRepo domainInfrastructure.InfrastructureRepository
}

func NewScheduler(infraRepo domainInfrastructure.InfrastructureRepository) *Scheduler {
	return &Scheduler{infraRepo: infraRepo}
}

// ScoredNode represents a candidate node with a deterministic score.
type ScoredNode struct {
	Node  *domainInfrastructure.Node
	Score float64
}

// SelectAndReserve deterministically finds the most suitable node and atomically creates a resource reservation.
func (s *Scheduler) SelectAndReserve(ctx context.Context, req ScheduleRequirements) (*domainInfrastructure.ResourceReservation, *domainInfrastructure.Node, error) {
	if req.TTL <= 0 {
		req.TTL = 15 * time.Minute
	}

	// 1. Fetch active candidates
	var nodes []*domainInfrastructure.Node
	var err error

	if req.NodeGroupID != nil && *req.NodeGroupID != uuid.Nil {
		nodes, err = s.infraRepo.ListNodesByNodeGroup(ctx, *req.NodeGroupID)
	} else {
		nodes, err = s.infraRepo.ListActiveNodes(ctx)
	}
	if err != nil {
		return nil, nil, err
	}

	// 2. Filter nodes with sufficient capacity
	var candidates []ScoredNode
	for _, n := range nodes {
		if n.Status != "active" {
			continue
		}
		if n.AvailableCPU() < req.CPUCores {
			continue
		}
		if n.AvailableMemoryMB() < req.MemoryMB {
			continue
		}
		if n.AvailableDiskGB() < req.DiskGB {
			continue
		}

		// Calculate deterministic score:
		// Base score from remaining memory ratio and cpu ratio, scaled by node weight
		cpuRatio := n.AvailableCPU() / n.CPUTotal
		memRatio := float64(n.AvailableMemoryMB()) / float64(n.MemoryTotalMB)
		diskRatio := float64(n.AvailableDiskGB()) / float64(n.DiskTotalGB)
		avgFreeRatio := (cpuRatio + memRatio + diskRatio) / 3.0
		score := float64(n.Weight)*1000.0 + avgFreeRatio*100.0

		candidates = append(candidates, ScoredNode{
			Node:  n,
			Score: score,
		})
	}

	if len(candidates) == 0 {
		return nil, nil, domainInfrastructure.ErrNoSchedulableNode
	}

	// 3. Deterministically sort candidates:
	// Descending by Score; tie-breaker: ascending by Node ID string
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Node.ID.String() < candidates[j].Node.ID.String()
	})

	// 4. Try reserving on the best candidate, falling back to next candidate if a race occurred
	expiresAt := time.Now().UTC().Add(req.TTL)

	for _, cand := range candidates {
		res := &domainInfrastructure.ResourceReservation{
			ID:           uuid.New(),
			NodeID:       cand.Node.ID,
			OperationID:  req.OperationID,
			CPUCores:     req.CPUCores,
			MemoryMB:     req.MemoryMB,
			DiskGB:       req.DiskGB,
			IPv4Count:    req.IPv4Count,
			IPv6Count:    req.IPv6Count,
			NATPortCount: req.NATPortCount,
			Status:       "reserved",
			ExpiresAt:    expiresAt,
		}

		createdRes, err := s.infraRepo.ReserveResources(ctx, res)
		if err != nil {
			if errors.Is(err, domainInfrastructure.ErrResourceExhausted) ||
				errors.Is(err, domainInfrastructure.ErrNodeOffline) {
				// Concurrently filled up or node went offline; try next best candidate
				continue
			}
			return nil, nil, err
		}

		return createdRes, cand.Node, nil
	}

	return nil, nil, domainInfrastructure.ErrNoSchedulableNode
}
