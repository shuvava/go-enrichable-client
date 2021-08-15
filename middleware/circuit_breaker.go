package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/shuvava/go-enrichable-client/client"
)

// implements the Circuit Breaker pattern.
// See https://msdn.microsoft.com/en-us/library/dn589784.aspx.
// logic of this module taken from https://github.com/sony/gobreaker

// CircuitBreakerState is a type that represents a state of CircuitBreakerService.
type CircuitBreakerState int

// These constants are states of CircuitBreakerService.
const (
	CircuitBreakerStateClosed CircuitBreakerState = iota
	CircuitBreakerStateHalfOpen
	CircuitBreakerStateOpen
)

const defaultInterval = time.Duration(0) * time.Second
const defaultTimeout = time.Duration(60) * time.Second

var (
	// ErrTooManyRequests is returned when the CB state is half open and the requests count is over the cb maxRequests
	ErrTooManyRequests = errors.New("too many requests")
	// ErrOpenState is returned when the CB state is open
	ErrOpenState = errors.New("circuit breaker is open")
)

// String implements stringer interface.
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerStateClosed:
		return "closed"
	case CircuitBreakerStateHalfOpen:
		return "half-open"
	case CircuitBreakerStateOpen:
		return "open"
	default:
		return fmt.Sprintf("unknown state: %d", s)
	}
}

// CircuitBreakerCounts holds the numbers of requests and their successes/failures.
// CircuitBreakerService clears the internal CircuitBreakerCounts either
// on the change of the state or at the closed-state intervals.
// CircuitBreakerCounts ignores the results of the requests sent before clearing.
type CircuitBreakerCounts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

func (c *CircuitBreakerCounts) onRequest() {
	c.Requests++
}

func (c *CircuitBreakerCounts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *CircuitBreakerCounts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *CircuitBreakerCounts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// CircuitBreakerSettings configures CircuitBreakerService:
//
// MaxRequests is the maximum number of requests allowed to pass through
// when the CircuitBreakerService is half-open.
// If MaxRequests is 0, the CircuitBreakerService allows only 1 request.
//
// Interval is the cyclic period of the closed state
// for the CircuitBreakerService to clear the internal CircuitBreakerCounts.
// If Interval is less than or equal to 0, the CircuitBreakerService doesn't clear internal CircuitBreakerCounts during the closed state.
//
// Timeout is the period of the open state,
// after which the state of the CircuitBreakerService becomes half-open.
// If Timeout is less than or equal to 0, the timeout value of the CircuitBreakerService is set to 60 seconds.
//
// ReadyToTrip is called with a copy of CircuitBreakerCounts whenever a request fails in the closed state.
// If ReadyToTrip returns true, the CircuitBreakerService will be placed into the open state.
// If ReadyToTrip is nil, default ReadyToTrip is used.
// Default ReadyToTrip returns true when the number of consecutive failures is more than 5.
//
// OnStateChange is called whenever the state of the CircuitBreakerService changes.
//
// IsSuccessful is called with the error returned from the request, if not nil.
// If IsSuccessful returns false, the error is considered a failure, and is counted towards tripping the circuit breaker.
// If IsSuccessful returns true, the error will be returned to the caller without tripping the circuit breaker.
// If IsSuccessful is nil, default IsSuccessful is used, which returns false for all non-nil errors.
type CircuitBreakerSettings struct {
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts CircuitBreakerCounts) bool
	OnStateChange func(from CircuitBreakerState, to CircuitBreakerState)
	IsSuccessful  func(resp *http.Response, err error) bool
}

// CircuitBreakerService is a state machine to prevent sending requests that are likely to fail.
type CircuitBreakerService struct {
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts CircuitBreakerCounts) bool
	isSuccessful  func(resp *http.Response, err error) bool
	onStateChange func(from CircuitBreakerState, to CircuitBreakerState)

	mutex      sync.Mutex
	state      CircuitBreakerState
	generation uint64
	counts     CircuitBreakerCounts
	expiry     time.Time
}

// NewCircuitBreakerService returns a new CircuitBreakerService configured with the given CircuitBreakerSettings.
func NewCircuitBreakerService(st CircuitBreakerSettings) *CircuitBreakerService {

	cb := new(CircuitBreakerService)

	cb.onStateChange = st.OnStateChange

	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}

	if st.Interval <= 0 {
		cb.interval = defaultInterval
	} else {
		cb.interval = st.Interval
	}

	if st.Timeout <= 0 {
		cb.timeout = defaultTimeout
	} else {
		cb.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}

	if st.IsSuccessful == nil {
		cb.isSuccessful = defaultIsSuccessful
	} else {
		cb.isSuccessful = st.IsSuccessful
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// Execute process http.Client Do operation
func (cb *CircuitBreakerService) Execute(_ *http.Client, next client.Responder) client.Responder {
	return func(request *http.Request) (*http.Response, error) {
		generation, err := cb.beforeRequest()
		if err != nil {
			return nil, err
		}

		result, err := next(request)

		cb.afterRequest(generation, cb.isSuccessful(result, err))
		return result, err
	}
}

// CircuitBreaker adds Circuit Breaker middleware to requests
func CircuitBreaker(c CircuitBreakerSettings) client.MiddlewareFunc {
	cb := NewCircuitBreakerService(c)
	return cb.Execute
}

func defaultReadyToTrip(counts CircuitBreakerCounts) bool {
	return counts.ConsecutiveFailures > 5
}

func defaultIsSuccessful(resp *http.Response, err error) bool {
	assertErr := client.AssertStatusCode(resp)
	return err == nil && assertErr == nil
}

// State returns the current state of the CircuitBreakerService.
func (cb *CircuitBreakerService) State() CircuitBreakerState {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns internal counters
func (cb *CircuitBreakerService) Counts() CircuitBreakerCounts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

func (cb *CircuitBreakerService) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == CircuitBreakerStateOpen {
		return generation, ErrOpenState
	} else if state == CircuitBreakerStateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.onRequest()
	return generation, nil
}

func (cb *CircuitBreakerService) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreakerService) onSuccess(state CircuitBreakerState, now time.Time) {
	switch state {
	case CircuitBreakerStateClosed:
		cb.counts.onSuccess()
	case CircuitBreakerStateHalfOpen:
		cb.counts.onSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(CircuitBreakerStateClosed, now)
		}
	}
}

func (cb *CircuitBreakerService) onFailure(state CircuitBreakerState, now time.Time) {
	switch state {
	case CircuitBreakerStateClosed:
		cb.counts.onFailure()
		if cb.readyToTrip(cb.counts) {
			cb.setState(CircuitBreakerStateOpen, now)
		}
	case CircuitBreakerStateHalfOpen:
		cb.setState(CircuitBreakerStateOpen, now)
	}
}

func (cb *CircuitBreakerService) currentState(now time.Time) (CircuitBreakerState, uint64) {
	switch cb.state {
	case CircuitBreakerStateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case CircuitBreakerStateOpen:
		if cb.expiry.Before(now) {
			cb.setState(CircuitBreakerStateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreakerService) setState(state CircuitBreakerState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(prev, state)
	}
}

func (cb *CircuitBreakerService) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.clear()

	var zero time.Time
	switch cb.state {
	case CircuitBreakerStateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case CircuitBreakerStateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // CircuitBreakerStateHalfOpen
		cb.expiry = zero
	}
}
