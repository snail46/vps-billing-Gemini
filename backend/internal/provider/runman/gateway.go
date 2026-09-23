package runman

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"vps-billing/internal/provider"
)

type CommandHandler func(cmd AgentCommand) (*AgentCommandResult, error)

type AgentSession struct {
	NodeID         string
	AuthToken      string
	ConnectedAt    time.Time
	LastHeartbeat  time.Time
	LastPayload    *HeartbeatPayload
	PortForwards   map[string][]PortForwardRule // instanceID -> rules
	commandHandler CommandHandler
}

type Gateway struct {
	mu           sync.RWMutex
	agents       map[string]*AgentSession
	validTokens  map[string]string // nodeID -> expectedToken
	heartbeatTTL time.Duration
}

func NewGateway(heartbeatTTL time.Duration) *Gateway {
	if heartbeatTTL <= 0 {
		heartbeatTTL = 30 * time.Second
	}
	return &Gateway{
		agents:       make(map[string]*AgentSession),
		validTokens:  make(map[string]string),
		heartbeatTTL: heartbeatTTL,
	}
}

func (g *Gateway) SetNodeToken(nodeID, token string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.validTokens[nodeID] = token
}

func (g *Gateway) RegisterAgent(nodeID, token string, handler CommandHandler) (*AgentSession, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	expectedToken, hasToken := g.validTokens[nodeID]
	if hasToken && expectedToken != token {
		return nil, &provider.Error{
			Code:       provider.ErrCodeProviderAuthFailed,
			Provider:   "runman",
			RawMessage: "invalid agent authentication token",
		}
	}

	session := &AgentSession{
		NodeID:         nodeID,
		AuthToken:      token,
		ConnectedAt:    time.Now(),
		LastHeartbeat:  time.Now(),
		PortForwards:   make(map[string][]PortForwardRule),
		commandHandler: handler,
	}
	g.agents[nodeID] = session
	return session, nil
}

func (g *Gateway) UnregisterAgent(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.agents, nodeID)
}

func (g *Gateway) HandleHeartbeat(nodeID string, payload HeartbeatPayload) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	session, exists := g.agents[nodeID]
	if !exists {
		return &provider.Error{
			Code:       provider.ErrCodeNodeOffline,
			Provider:   "runman",
			RawMessage: fmt.Sprintf("node %s is not registered", nodeID),
		}
	}

	session.LastHeartbeat = time.Now()
	session.LastPayload = &payload
	return nil
}

func (g *Gateway) GetAgentSession(nodeID string) (*AgentSession, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	session, exists := g.agents[nodeID]
	if !exists {
		return nil, &provider.Error{
			Code:       provider.ErrCodeNodeOffline,
			Provider:   "runman",
			RawMessage: fmt.Sprintf("node %s has no active agent session", nodeID),
		}
	}

	if time.Since(session.LastHeartbeat) > g.heartbeatTTL {
		return nil, &provider.Error{
			Code:       provider.ErrCodeNodeOffline,
			Provider:   "runman",
			RawMessage: fmt.Sprintf("agent for node %s heartbeat expired", nodeID),
		}
	}

	return session, nil
}

func (g *Gateway) DispatchCommand(ctx context.Context, cmd AgentCommand) (*AgentCommandResult, error) {
	session, err := g.GetAgentSession(cmd.NodeID)
	if err != nil {
		return nil, err
	}

	if session.commandHandler == nil {
		return nil, &provider.Error{
			Code:       provider.ErrCodeProviderUnavailable,
			Provider:   "runman",
			RawMessage: "agent session does not have command handler attached",
		}
	}

	timeout := time.Duration(cmd.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan struct {
		res *AgentCommandResult
		err error
	}, 1)

	go func() {
		res, err := session.commandHandler(cmd)
		done <- struct {
			res *AgentCommandResult
			err error
		}{res, err}
	}()

	select {
	case <-cmdCtx.Done():
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return nil, &provider.Error{
				Code:       provider.ErrCodeProviderTimeout,
				Retryable:  true,
				Provider:   "runman",
				RawMessage: fmt.Sprintf("command %s to node %s timed out", cmd.Action, cmd.NodeID),
			}
		}
		return nil, cmdCtx.Err()
	case r := <-done:
		if r.err != nil {
			return nil, r.err
		}
		if !r.res.Success {
			code := r.res.ErrorCode
			if code == "" {
				code = provider.ErrCodeUnknown
			}
			return nil, &provider.Error{
				Code:       code,
				Provider:   "runman",
				RawMessage: r.res.ErrorMessage,
			}
		}
		return r.res, nil
	}
}

func (g *Gateway) ActiveAgentCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	count := 0
	now := time.Now()
	for _, a := range g.agents {
		if now.Sub(a.LastHeartbeat) <= g.heartbeatTTL {
			count++
		}
	}
	return count
}
