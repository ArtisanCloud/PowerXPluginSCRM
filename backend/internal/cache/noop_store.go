package cache

import (
	"context"
	"time"
)

type NoopStore struct{}

func NewNoopStore() *NoopStore {
	return &NoopStore{}
}

func (n *NoopStore) Get(ctx context.Context, key string) ([]byte, bool) {
	return nil, false
}

func (n *NoopStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (n *NoopStore) Delete(ctx context.Context, key string) error {
	return nil
}

func (n *NoopStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (n *NoopStore) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, nil
}
