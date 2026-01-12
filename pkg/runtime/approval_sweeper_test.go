package runtime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type testExpirer struct {
	calls    int64
	deadline int64
	ch       chan struct{}
}

func (t *testExpirer) ExpireApprovals(ctx context.Context) (int, error) {
	atomic.AddInt64(&t.calls, 1)
	if deadline, ok := ctx.Deadline(); ok {
		atomic.StoreInt64(&t.deadline, deadline.UnixNano())
	}
	select {
	case t.ch <- struct{}{}:
	default:
	}
	return 0, nil
}

func TestApprovalSweeperTimeout(t *testing.T) {
	expirer := &testExpirer{ch: make(chan struct{}, 1)}
	rt := NewLocal()
	rt.AddApprovalExpirer(expirer)
	rt.SetApprovalSweepInterval(10 * time.Millisecond)
	rt.SetApprovalSweepTimeout(50 * time.Millisecond)

	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() {
		_ = rt.Stop(context.Background())
	}()

	select {
	case <-expirer.ch:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected sweeper call")
	}

	if atomic.LoadInt64(&expirer.calls) == 0 {
		t.Fatalf("expected expirer to be called")
	}
	if atomic.LoadInt64(&expirer.deadline) == 0 {
		t.Fatalf("expected deadline to be set on sweep context")
	}
}
