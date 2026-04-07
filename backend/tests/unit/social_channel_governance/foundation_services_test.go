package social_channel_governance_test

import (
	"context"
	"sync/atomic"
	"testing"

	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
)

type stubAdapter struct{}

func (stubAdapter) CapabilityMatrix(ctx context.Context, tenantUUID string) map[string]string {
	return map[string]string{
		"auth": "supported",
		"org":  "partial",
	}
}

func TestChannelFactoryResolve(t *testing.T) {
	factory := socialsvc.NewChannelFactory()
	if err := factory.Register("wechat", "wecom", stubAdapter{}); err != nil {
		t.Fatalf("register adapter failed: %v", err)
	}
	if _, ok := factory.Resolve("wechat", "wecom"); !ok {
		t.Fatalf("expected adapter resolved")
	}
}

func TestIdempotencyKeyStable(t *testing.T) {
	svc := socialsvc.NewIdempotencyService()
	k1 := svc.BuildKey("Tenant-A", "Org", "Pull")
	k2 := svc.BuildKey(" tenant-a ", "org", "pull")
	if k1 != k2 {
		t.Fatalf("expected stable key, got %s vs %s", k1, k2)
	}
}

func TestSyncSchedulerSerializesSameScope(t *testing.T) {
	scheduler := socialsvc.NewSyncScheduler()
	var active int32
	unlock := scheduler.Lock("tenant-1", "org")
	done := make(chan struct{})
	go func() {
		defer close(done)
		unlock2 := scheduler.Lock("tenant-1", "org")
		atomic.StoreInt32(&active, 1)
		unlock2()
	}()
	if atomic.LoadInt32(&active) != 0 {
		t.Fatalf("lock leaked; second goroutine should still be blocked")
	}
	unlock()
	<-done
	if atomic.LoadInt32(&active) != 1 {
		t.Fatalf("second goroutine did not enter critical section")
	}
}
