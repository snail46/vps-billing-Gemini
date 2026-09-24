package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	UserSessionCookieName  = "vps_user_session"
	AdminSessionCookieName = "vps_admin_session"
	CSRFCookieName         = "vps_csrf_token"
	UserCSRFCookieName     = "vps_csrf_token_user"
	AdminCSRFCookieName    = "vps_csrf_token_admin"

	UserSessionTTL  = 7 * 24 * time.Hour
	AdminSessionTTL = 24 * time.Hour
)

type Session struct {
	Token       string    `json:"token"`
	ActorType   string    `json:"actor_type"` // "user" or "admin"
	ActorID     uuid.UUID `json:"actor_id"`
	Email       string    `json:"email"`
	CSRFToken   string    `json:"csrf_token"`
	Roles       []string  `json:"roles,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type Store interface {
	Set(ctx context.Context, s *Session, ttl time.Duration) error
	Get(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
}

// RedisSessionStore implements Store using Redis
type RedisSessionStore struct {
	client *redis.Client
}

func NewRedisSessionStore(client *redis.Client) *RedisSessionStore {
	return &RedisSessionStore{client: client}
}

func (s *RedisSessionStore) Set(ctx context.Context, sess *Session, ttl time.Duration) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	key := fmt.Sprintf("session:%s:%s", sess.ActorType, sess.Token)
	return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *RedisSessionStore) Get(ctx context.Context, token string) (*Session, error) {
	// Try user session first, then admin session
	for _, actorType := range []string{"user", "admin"} {
		key := fmt.Sprintf("session:%s:%s", actorType, token)
		data, err := s.client.Get(ctx, key).Bytes()
		if err == nil {
			var sess Session
			if err := json.Unmarshal(data, &sess); err == nil {
				return &sess, nil
			}
		}
	}
	return nil, errors.New("session not found")
}

func (s *RedisSessionStore) Delete(ctx context.Context, token string) error {
	for _, actorType := range []string{"user", "admin"} {
		key := fmt.Sprintf("session:%s:%s", actorType, token)
		_ = s.client.Del(ctx, key).Err()
	}
	return nil
}

// MemorySessionStore implements Store in-memory for testing
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *MemorySessionStore) Set(_ context.Context, sess *Session, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.Token] = sess
	return nil
}

func (s *MemorySessionStore) Get(_ context.Context, token string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, exists := s.sessions[token]
	if !exists {
		return nil, errors.New("session not found")
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, errors.New("session expired")
	}
	return sess, nil
}

func (s *MemorySessionStore) Delete(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

// SessionManager provides session orchestration
type Manager struct {
	store    Store
	isSecure bool
}

func NewManager(store Store, isSecure bool) *Manager {
	return &Manager{
		store:    store,
		isSecure: isSecure,
	}
}

func GenerateSecureToken(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *Manager) CreateUserSession(ctx context.Context, userID uuid.UUID, email string) (*Session, error) {
	token, err := GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}
	csrfToken, err := GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sess := &Session{
		Token:     token,
		ActorType: "user",
		ActorID:   userID,
		Email:     email,
		CSRFToken: csrfToken,
		CreatedAt: now,
		ExpiresAt: now.Add(UserSessionTTL),
	}

	if err := m.store.Set(ctx, sess, UserSessionTTL); err != nil {
		return nil, err
	}

	return sess, nil
}

func (m *Manager) CreateAdminSession(ctx context.Context, adminID uuid.UUID, email string, roles []string, perms []string) (*Session, error) {
	token, err := GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}
	csrfToken, err := GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sess := &Session{
		Token:       token,
		ActorType:   "admin",
		ActorID:     adminID,
		Email:       email,
		CSRFToken:   csrfToken,
		Roles:       roles,
		Permissions: perms,
		CreatedAt:   now,
		ExpiresAt:   now.Add(AdminSessionTTL),
	}

	if err := m.store.Set(ctx, sess, AdminSessionTTL); err != nil {
		return nil, err
	}

	return sess, nil
}

func (m *Manager) GetSession(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, errors.New("empty session token")
	}
	return m.store.Get(ctx, token)
}

func (m *Manager) DestroySession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return m.store.Delete(ctx, token)
}

func (m *Manager) SetSessionCookie(w http.ResponseWriter, cookieName, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   m.isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) ClearSessionCookie(w http.ResponseWriter, cookieName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) SetCSRFCookie(w http.ResponseWriter, csrfToken string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: false, // Frontend JavaScript must be able to read this to set X-CSRF-Token header
		Secure:   m.isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) SetActorCSRFCookie(w http.ResponseWriter, cookieName string, csrfToken string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    csrfToken,
		Path:     "/",
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: false,
		Secure:   m.isSecure,
		SameSite: http.SameSiteLaxMode,
	})
	// Also set the general CSRFCookieName for backward compatibility with tests/clients
	m.SetCSRFCookie(w, csrfToken, ttl)
}

func (m *Manager) ClearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   m.isSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) ClearActorCSRFCookie(w http.ResponseWriter, cookieName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   m.isSecure,
		SameSite: http.SameSiteLaxMode,
	})
	m.ClearCSRFCookie(w)
}
