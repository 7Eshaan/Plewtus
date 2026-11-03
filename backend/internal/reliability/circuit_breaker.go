package reliability

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the state of a circuit breaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when a request is rejected because the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open: provider temporarily unavailable")

// BreakerOptions configures a CircuitBreaker.
type BreakerOptions struct {
	FailureThreshold int           // Number of consecutive failures to trip circuit (default 3)
	Cooldown         time.Duration // Time to wait in open state before trial probe (default 30s)
}

// CircuitBreaker protects upstream providers from being overwhelmed during failures.
type CircuitBreaker struct {
	mu               sync.Mutex
	name             string
	failureThreshold int
	cooldown         time.Duration
	state            State
	failures         int
	lastTrippedAt    time.Time
	halfOpenProbing  bool
}

// NewCircuitBreaker creates a new circuit breaker with given options.
func NewCircuitBreaker(name string, opts BreakerOptions) *CircuitBreaker {
	if opts.FailureThreshold <= 0 {
		opts.FailureThreshold = 3
	}
	if opts.Cooldown <= 0 {
		opts.Cooldown = 30 * time.Second
	}

	return &CircuitBreaker{
		name:             name,
		failureThreshold: opts.FailureThreshold,
		cooldown:         opts.Cooldown,
		state:            StateClosed,
	}
}

// Name returns the provider name for this breaker.
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.updateStateLocked(time.Now())
	return cb.state
}

// Allow checks if a request is permitted. If the circuit is open, returns ErrCircuitOpen.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.updateStateLocked(now)

	switch cb.state {
	case StateClosed:
		return nil

	case StateHalfOpen:
		if !cb.halfOpenProbing {
			// Allow 1 probe request through
			cb.halfOpenProbing = true
			return nil
		}
		// Probe is already in-flight, reject concurrent calls until probe completes
		return fmt.Errorf("%w (%s in probe)", ErrCircuitOpen, cb.name)

	case StateOpen:
		remaining := cb.cooldown - now.Sub(cb.lastTrippedAt)
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Errorf("%w (%s cooling down, %.1fs remaining)", ErrCircuitOpen, cb.name, remaining.Seconds())

	default:
		return nil
	}
}

// ReportSuccess records a successful provider execution, closing the circuit.
func (cb *CircuitBreaker) ReportSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = StateClosed
	cb.halfOpenProbing = false
}

// ReportFailure records a failed provider execution.
func (cb *CircuitBreaker) ReportFailure(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateHalfOpen:
		// Probe failed, immediately trip back to Open
		cb.state = StateOpen
		cb.lastTrippedAt = now
		cb.halfOpenProbing = false

	case StateClosed:
		cb.failures++
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastTrippedAt = now
		}

	case StateOpen:
		// Already open, refresh timestamp
		cb.lastTrippedAt = now
	}
}

// Reset restores the breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenProbing = false
}

func (cb *CircuitBreaker) updateStateLocked(now time.Time) {
	if cb.state == StateOpen {
		if now.Sub(cb.lastTrippedAt) >= cb.cooldown {
			cb.state = StateHalfOpen
			cb.halfOpenProbing = false
		}
	}
}

// Manager coordinates circuit breakers across all search providers.
type Manager struct {
	mu       sync.RWMutex
	options  BreakerOptions
	breakers map[string]*CircuitBreaker
}

// NewManager creates a manager for provider circuit breakers.
func NewManager(opts BreakerOptions) *Manager {
	return &Manager{
		options:  opts,
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Get returns the circuit breaker for a provider, creating one if not exists.
func (m *Manager) Get(name string) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	cb, exists := m.breakers[name]
	if !exists {
		cb = NewCircuitBreaker(name, m.options)
		m.breakers[name] = cb
	}
	return cb
}

// IsHealthy returns true if the provider's circuit is not Open.
func (m *Manager) IsHealthy(name string) bool {
	cb := m.Get(name)
	return cb.State() != StateOpen
}

// ResetAll resets all circuit breakers to closed state.
func (m *Manager) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, cb := range m.breakers {
		cb.Reset()
	}
}
