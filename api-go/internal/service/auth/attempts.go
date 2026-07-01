package auth

import (
	"sync"
	"time"
)

const (
	maxCodeAttempts = 5
	attemptsTTL     = 24 * time.Hour
)

type attemptsEntry struct {
	count    int
	lastSeen time.Time
}

type attemptsStore struct {
	mu      sync.Mutex
	entries map[string]*attemptsEntry
}

var codeAttempts = &attemptsStore{
	entries: make(map[string]*attemptsEntry),
}

func init() {
	go func() {
		ticker := time.NewTicker(attemptsTTL)
		defer ticker.Stop()
		for range ticker.C {
			codeAttempts.evictStale()
			resendCooldowns.evictStale()
		}
	}()
}

func (s *attemptsStore) evictStale() {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-attemptsTTL)
	for k, e := range s.entries {
		if e.lastSeen.Before(cutoff) {
			delete(s.entries, k)
		}
	}
}

// record increments the counter and returns true if the attempt is allowed.
func (s *attemptsStore) record(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entries[key]
	if !ok {
		e = &attemptsEntry{}
		s.entries[key] = e
	}
	e.lastSeen = time.Now()
	e.count++
	return e.count <= maxCodeAttempts
}

// reset clears the counter for a key (call when a new code is issued).
func (s *attemptsStore) reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
}

// ResetAllAttempts clears all counters (for testing).
func ResetAllAttempts() {
	codeAttempts.mu.Lock()
	defer codeAttempts.mu.Unlock()
	codeAttempts.entries = make(map[string]*attemptsEntry)
}

// ResetResendCooldowns clears all resend cooldown timestamps (for testing).
func ResetResendCooldowns() {
	resendCooldowns.mu.Lock()
	defer resendCooldowns.mu.Unlock()
	resendCooldowns.lastAt = make(map[string]time.Time)
}

func verifyAttemptKey(email string) string  { return "verify:" + email }
func resetAttemptKey(email string) string   { return "reset:" + email }
func resendCooldownKey(email string) string { return "resend:" + email }

type cooldownStore struct {
	mu       sync.Mutex
	lastAt   map[string]time.Time
	cooldown time.Duration
}

var resendCooldowns = &cooldownStore{
	lastAt:   make(map[string]time.Time),
	cooldown: 60 * time.Second,
}

// SetResendCooldown overrides the cooldown duration (call once during service init).
func SetResendCooldown(d time.Duration) {
	resendCooldowns.mu.Lock()
	defer resendCooldowns.mu.Unlock()
	resendCooldowns.cooldown = d
}

// allow returns true and records the timestamp if the cooldown has elapsed.
// Returns false if called again before cooldown expires.
// When cooldown is 0, always allows (useful for tests).
func (s *cooldownStore) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cooldown == 0 {
		return true
	}
	if t, ok := s.lastAt[key]; ok && time.Since(t) < s.cooldown {
		return false
	}
	s.lastAt[key] = time.Now()
	return true
}

// evictStale drops timestamps older than the cooldown window; after it
// elapses the entry no longer affects allow(), so it can be safely removed.
func (s *cooldownStore) evictStale() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cooldown == 0 {
		s.lastAt = make(map[string]time.Time)
		return
	}
	cutoff := time.Now().Add(-s.cooldown)
	for k, t := range s.lastAt {
		if t.Before(cutoff) {
			delete(s.lastAt, k)
		}
	}
}
