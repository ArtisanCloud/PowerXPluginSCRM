package social_channel_governance

import (
	"strings"
	"sync"
)

type SyncScheduler struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewSyncScheduler() *SyncScheduler {
	return &SyncScheduler{locks: map[string]*sync.Mutex{}}
}

func (s *SyncScheduler) Lock(tenantUUID, domain string) func() {
	if s == nil {
		return func() {}
	}
	key := strings.ToLower(strings.TrimSpace(tenantUUID)) + ":" + strings.TrimSpace(domain)
	s.mu.Lock()
	locker, ok := s.locks[key]
	if !ok {
		locker = &sync.Mutex{}
		s.locks[key] = locker
	}
	s.mu.Unlock()
	locker.Lock()
	return locker.Unlock
}
