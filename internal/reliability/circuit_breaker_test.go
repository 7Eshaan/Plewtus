package reliability

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	opts := BreakerOptions{
		FailureThreshold: 2,
		Cooldown:         50 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test-engine", opts)

	// Initially closed
	if cb.State() != StateClosed {
		t.Fatalf("expected state Closed, got %v", cb.State())
	}
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected Allow() nil, got %v", err)
	}

	// 1st failure - still closed (threshold is 2)
	cb.ReportFailure(errors.New("fail 1"))
	if cb.State() != StateClosed {
		t.Fatalf("expected state Closed after 1 failure, got %v", cb.State())
	}
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected Allow() nil, got %v", err)
	}

	// 2nd failure - trips to Open
	cb.ReportFailure(errors.New("fail 2"))
	if cb.State() != StateOpen {
		t.Fatalf("expected state Open after 2 failures, got %v", cb.State())
	}

	// In Open state, Allow() is rejected
	err := cb.Allow()
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	// Wait for cooldown to expire
	time.Sleep(60 * time.Millisecond)

	// State should now be Half-Open
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state HalfOpen after cooldown, got %v", cb.State())
	}

	// First probe request allowed in Half-Open
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected probe Allow() nil, got %v", err)
	}

	// Concurrent requests while probing should be rejected
	if err := cb.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected concurrent probe rejected with ErrCircuitOpen, got %v", err)
	}

	// Probe succeeds -> transitions to Closed
	cb.ReportSuccess()
	if cb.State() != StateClosed {
		t.Fatalf("expected state Closed after probe success, got %v", cb.State())
	}
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected Allow() nil after recovery, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenProbeFailureTripsToOpen(t *testing.T) {
	opts := BreakerOptions{
		FailureThreshold: 1,
		Cooldown:         40 * time.Millisecond,
	}
	cb := NewCircuitBreaker("test-engine-2", opts)

	// Trip to Open
	cb.ReportFailure(errors.New("initial fail"))
	if cb.State() != StateOpen {
		t.Fatalf("expected Open state, got %v", cb.State())
	}

	time.Sleep(50 * time.Millisecond)

	// Should be Half-Open
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected HalfOpen state, got %v", cb.State())
	}

	// Allow probe
	if err := cb.Allow(); err != nil {
		t.Fatalf("expected probe allowed, got %v", err)
	}

	// Probe fails -> trips immediately back to Open
	cb.ReportFailure(errors.New("probe failed"))
	if cb.State() != StateOpen {
		t.Fatalf("expected state Open after failed probe, got %v", cb.State())
	}
}

func TestCircuitBreaker_Concurrency(t *testing.T) {
	opts := BreakerOptions{
		FailureThreshold: 10,
		Cooldown:         100 * time.Millisecond,
	}
	cb := NewCircuitBreaker("concurrent-engine", opts)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = cb.Allow()
				if j%2 == 0 {
					cb.ReportSuccess()
				} else {
					cb.ReportFailure(errors.New("mock error"))
				}
				_ = cb.State()
			}
		}(i)
	}
	wg.Wait()
}

func TestManager(t *testing.T) {
	mgr := NewManager(BreakerOptions{
		FailureThreshold: 2,
		Cooldown:         1 * time.Second,
	})

	if !mgr.IsHealthy("duckduckgo") {
		t.Fatal("expected duckduckgo initially healthy")
	}

	ddgBreaker := mgr.Get("duckduckgo")
	ddgBreaker.ReportFailure(errors.New("f1"))
	ddgBreaker.ReportFailure(errors.New("f2"))

	if mgr.IsHealthy("duckduckgo") {
		t.Fatal("expected duckduckgo unhealthy after 2 failures")
	}

	if !mgr.IsHealthy("bing") {
		t.Fatal("expected bing to remain healthy")
	}

	mgr.ResetAll()
	if !mgr.IsHealthy("duckduckgo") {
		t.Fatal("expected duckduckgo healthy after reset")
	}
}
