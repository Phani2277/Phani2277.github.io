package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	errUnauthorized = errors.New("unauthorized")
	sessionStore    = newInMemorySessionStore()
)

type sessionData struct {
	UserID    int64
	ExpiresAt time.Time
}

type inMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]sessionData
}

func newInMemorySessionStore() *inMemorySessionStore {
	return &inMemorySessionStore{
		sessions: make(map[string]sessionData),
	}
}

func (s *inMemorySessionStore) create(userID int64, ttl time.Duration) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = sessionData{
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}
	return token, nil
}

func (s *inMemorySessionStore) getUserID(token string) (int64, bool) {
	s.mu.RLock()
	data, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return 0, false
	}

	if time.Now().After(data.ExpiresAt) {
		s.mu.Lock()
		delete(s.sessions, token)
		s.mu.Unlock()
		return 0, false
	}
	return data.UserID, true
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func bearerTokenFromRequest(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func requireAuthUserID(r *http.Request) (int64, error) {
	token := bearerTokenFromRequest(r)
	if token == "" {
		return 0, errUnauthorized
	}
	userID, ok := sessionStore.getUserID(token)
	if !ok {
		return 0, errUnauthorized
	}
	return userID, nil
}
