package cache

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

type memoryEntry struct {
	value []byte
	exp   time.Time
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]memoryEntry
	cfg   *config.CacheConfig
}

func NewMemoryStore(cfg *config.CacheConfig) *MemoryStore {
	return &MemoryStore{
		items: make(map[string]memoryEntry),
		cfg:   cfg,
	}
}

func (m *MemoryStore) Get(ctx context.Context, key string) ([]byte, bool) {
	k := buildKey(m.prefix(), key)
	m.mu.RLock()
	entry, ok := m.items[k]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !entry.exp.IsZero() && time.Now().After(entry.exp) {
		m.mu.Lock()
		delete(m.items, k)
		m.mu.Unlock()
		return nil, false
	}
	return entry.value, true
}

func (m *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	k := buildKey(m.prefix(), key)
	exp := time.Time{}
	ttl = normalizeTTL(m.cfg, ttl)
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.items[k] = memoryEntry{value: value, exp: exp}
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) Delete(ctx context.Context, key string) error {
	k := buildKey(m.prefix(), key)
	m.mu.Lock()
	delete(m.items, k)
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	k := buildKey(m.prefix(), key)
	m.mu.RLock()
	entry, ok := m.items[k]
	m.mu.RUnlock()
	if !ok {
		return 0, errors.New("cache key not found")
	}
	if entry.exp.IsZero() {
		return 0, nil
	}
	ttl := time.Until(entry.exp)
	if ttl < 0 {
		return 0, errors.New("cache key expired")
	}
	return ttl, nil
}

func (m *MemoryStore) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	k := buildKey(m.prefix(), key)
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.items[k]
	if !entry.exp.IsZero() && time.Now().After(entry.exp) {
		entry = memoryEntry{}
	}
	var current int64
	if len(entry.value) > 0 {
		v, err := strconv.ParseInt(string(entry.value), 10, 64)
		if err != nil {
			return 0, err
		}
		current = v
	}
	current += delta
	entry.value = []byte(strconv.FormatInt(current, 10))
	if entry.exp.IsZero() {
		ttl := normalizeTTL(m.cfg, 0)
		if ttl > 0 {
			entry.exp = time.Now().Add(ttl)
		}
	}
	m.items[k] = entry
	return current, nil
}

func (m *MemoryStore) prefix() string {
	if m.cfg == nil {
		return ""
	}
	return m.cfg.Prefix
}
