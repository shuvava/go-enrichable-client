package middleware

import (
	"fmt"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type StateChange struct {
	from CircuitBreakerState
	to   CircuitBreakerState
}

var stateChange StateChange

func pseudoSleep(cb *CircuitBreakerService, period time.Duration) {
	if !cb.expiry.IsZero() {
		cb.expiry = cb.expiry.Add(-period)
	}
}

func succeed(cb *CircuitBreakerService) error {
	fn := cb.Execute(http.DefaultClient, func(req *http.Request) (*http.Response, error) {
		return nil, nil
	})
	_, err := fn(nil)
	return err
}

func succeedLater(cb *CircuitBreakerService, delay time.Duration) <-chan error {
	ch := make(chan error)
	go func() {
		fn := cb.Execute(http.DefaultClient, func(req *http.Request) (*http.Response, error) {
			time.Sleep(delay)
			return nil, nil
		})
		_, err := fn(nil)
		ch <- err
	}()
	return ch
}

func fail(cb *CircuitBreakerService) error {
	msg := "fail"
	fn := cb.Execute(http.DefaultClient, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf(msg)
	})
	_, err := fn(nil)
	if err.Error() == msg {
		return nil
	}
	return err
}

func TestStateConstants(t *testing.T) {
	assert.Equal(t, CircuitBreakerState(0), CircuitBreakerStateClosed)
	assert.Equal(t, CircuitBreakerState(1), CircuitBreakerStateHalfOpen)
	assert.Equal(t, CircuitBreakerState(2), CircuitBreakerStateOpen)

	assert.Equal(t, CircuitBreakerStateClosed.String(), "closed")
	assert.Equal(t, CircuitBreakerStateHalfOpen.String(), "half-open")
	assert.Equal(t, CircuitBreakerStateOpen.String(), "open")
	assert.Equal(t, CircuitBreakerState(100).String(), "unknown state: 100")
}

func TestNewCircuitBreaker(t *testing.T) {
	defaultCB := NewCircuitBreakerService(CircuitBreakerSettings{})
	assert.Equal(t, uint32(1), defaultCB.maxRequests)
	assert.Equal(t, time.Duration(0), defaultCB.interval)
	assert.Equal(t, time.Duration(60)*time.Second, defaultCB.timeout)
	assert.NotNil(t, defaultCB.readyToTrip)
	assert.Nil(t, defaultCB.onStateChange)
	assert.Equal(t, CircuitBreakerStateClosed, defaultCB.state)
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, defaultCB.counts)
	assert.True(t, defaultCB.expiry.IsZero())

	customCB := newCustom()
	assert.Equal(t, uint32(3), customCB.maxRequests)
	assert.Equal(t, time.Duration(30)*time.Second, customCB.interval)
	assert.Equal(t, time.Duration(90)*time.Second, customCB.timeout)
	assert.NotNil(t, customCB.readyToTrip)
	assert.NotNil(t, customCB.onStateChange)
	assert.Equal(t, CircuitBreakerStateClosed, customCB.state)
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, customCB.counts)
	assert.False(t, customCB.expiry.IsZero())

	negativeDurationCB := newNegativeDurationCB()
	assert.Equal(t, uint32(1), negativeDurationCB.maxRequests)
	assert.Equal(t, time.Duration(0)*time.Second, negativeDurationCB.interval)
	assert.Equal(t, time.Duration(60)*time.Second, negativeDurationCB.timeout)
	assert.NotNil(t, negativeDurationCB.readyToTrip)
	assert.Nil(t, negativeDurationCB.onStateChange)
	assert.Equal(t, CircuitBreakerStateClosed, negativeDurationCB.state)
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, negativeDurationCB.counts)
	assert.True(t, negativeDurationCB.expiry.IsZero())
}

func TestDefaultCircuitBreaker(t *testing.T) {
	defaultCB := NewCircuitBreakerService(CircuitBreakerSettings{})
	for i := 0; i < 5; i++ {
		assert.Nil(t, fail(defaultCB))
	}
	assert.Equal(t, CircuitBreakerStateClosed, defaultCB.State())
	assert.Equal(t, CircuitBreakerCounts{5, 0, 5, 0, 5}, defaultCB.counts)

	assert.Nil(t, succeed(defaultCB))
	assert.Equal(t, CircuitBreakerStateClosed, defaultCB.State())
	assert.Equal(t, CircuitBreakerCounts{6, 1, 5, 1, 0}, defaultCB.counts)

	assert.Nil(t, fail(defaultCB))
	assert.Equal(t, CircuitBreakerStateClosed, defaultCB.State())
	assert.Equal(t, CircuitBreakerCounts{7, 1, 6, 0, 1}, defaultCB.counts)

	// CircuitBreakerStateClosed to CircuitBreakerStateOpen
	for i := 0; i < 5; i++ {
		assert.Nil(t, fail(defaultCB)) // 6 consecutive failures
	}
	assert.Equal(t, CircuitBreakerStateOpen, defaultCB.State())

	assert.Error(t, succeed(defaultCB))
	assert.Error(t, fail(defaultCB))
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, defaultCB.counts)

	pseudoSleep(defaultCB, time.Duration(59)*time.Second)
	assert.Equal(t, CircuitBreakerStateOpen, defaultCB.State())

	// CircuitBreakerStateOpen to CircuitBreakerStateHalfOpen
	pseudoSleep(defaultCB, time.Duration(1)*time.Second) // over Timeout
	assert.Equal(t, CircuitBreakerStateHalfOpen, defaultCB.State())
	assert.True(t, defaultCB.expiry.IsZero())

	// CircuitBreakerStateHalfOpen to CircuitBreakerStateOpen
	assert.Nil(t, fail(defaultCB))
	assert.Equal(t, CircuitBreakerStateOpen, defaultCB.State())
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, defaultCB.counts)
	assert.False(t, defaultCB.expiry.IsZero())

	// CircuitBreakerStateOpen to CircuitBreakerStateHalfOpen
	pseudoSleep(defaultCB, time.Duration(60)*time.Second)
	assert.Equal(t, CircuitBreakerStateHalfOpen, defaultCB.State())
	assert.True(t, defaultCB.expiry.IsZero())

	// CircuitBreakerStateHalfOpen to CircuitBreakerStateClosed
	assert.Nil(t, succeed(defaultCB))
	assert.Equal(t, CircuitBreakerStateClosed, defaultCB.State())
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, defaultCB.counts)
	assert.True(t, defaultCB.expiry.IsZero())
}

func TestCustomCircuitBreaker(t *testing.T) {
	customCB := newCustom()

	for i := 0; i < 5; i++ {
		assert.Nil(t, succeed(customCB))
		assert.Nil(t, fail(customCB))
	}
	assert.Equal(t, CircuitBreakerStateClosed, customCB.State())
	assert.Equal(t, CircuitBreakerCounts{10, 5, 5, 0, 1}, customCB.counts)

	pseudoSleep(customCB, time.Duration(29)*time.Second)
	assert.Nil(t, succeed(customCB))
	assert.Equal(t, CircuitBreakerStateClosed, customCB.State())
	assert.Equal(t, CircuitBreakerCounts{11, 6, 5, 1, 0}, customCB.counts)

	pseudoSleep(customCB, time.Duration(1)*time.Second) // over Interval
	assert.Nil(t, fail(customCB))
	assert.Equal(t, CircuitBreakerStateClosed, customCB.State())
	assert.Equal(t, CircuitBreakerCounts{1, 0, 1, 0, 1}, customCB.counts)

	// CircuitBreakerStateClosed to CircuitBreakerStateOpen
	assert.Nil(t, succeed(customCB))
	assert.Nil(t, fail(customCB)) // failure ratio: 2/3 >= 0.6
	assert.Equal(t, CircuitBreakerStateOpen, customCB.State())
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, customCB.counts)
	assert.False(t, customCB.expiry.IsZero())
	assert.Equal(t, StateChange{CircuitBreakerStateClosed, CircuitBreakerStateOpen}, stateChange)

	// CircuitBreakerStateOpen to CircuitBreakerStateHalfOpen
	pseudoSleep(customCB, time.Duration(90)*time.Second)
	assert.Equal(t, CircuitBreakerStateHalfOpen, customCB.State())
	assert.True(t, customCB.expiry.IsZero())
	assert.Equal(t, StateChange{CircuitBreakerStateOpen, CircuitBreakerStateHalfOpen}, stateChange)

	assert.Nil(t, succeed(customCB))
	assert.Nil(t, succeed(customCB))
	assert.Equal(t, CircuitBreakerStateHalfOpen, customCB.State())
	assert.Equal(t, CircuitBreakerCounts{2, 2, 0, 2, 0}, customCB.counts)

	// CircuitBreakerStateHalfOpen to CircuitBreakerStateClosed
	ch := succeedLater(customCB, time.Duration(100)*time.Millisecond) // 3 consecutive successes
	time.Sleep(time.Duration(50) * time.Millisecond)
	assert.Equal(t, CircuitBreakerCounts{3, 2, 0, 2, 0}, customCB.counts)
	assert.Error(t, succeed(customCB)) // over MaxRequests
	assert.Nil(t, <-ch)
	assert.Equal(t, CircuitBreakerStateClosed, customCB.State())
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, customCB.counts)
	assert.False(t, customCB.expiry.IsZero())
	assert.Equal(t, StateChange{CircuitBreakerStateHalfOpen, CircuitBreakerStateClosed}, stateChange)
}

func TestGeneration(t *testing.T) {
	customCB := newCustom()
	pseudoSleep(customCB, time.Duration(29)*time.Second)
	assert.Nil(t, succeed(customCB))
	ch := succeedLater(customCB, time.Duration(1500)*time.Millisecond)
	time.Sleep(time.Duration(500) * time.Millisecond)
	assert.Equal(t, CircuitBreakerCounts{2, 1, 0, 1, 0}, customCB.counts)

	time.Sleep(time.Duration(500) * time.Millisecond) // over Interval
	assert.Equal(t, CircuitBreakerStateClosed, customCB.State())
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, customCB.counts)

	// the request from the previous generation has no effect on customCB.counts
	assert.Nil(t, <-ch)
	assert.Equal(t, CircuitBreakerCounts{0, 0, 0, 0, 0}, customCB.counts)
}

func TestCustomIsSuccessful(t *testing.T) {
	isSuccessful := func(*http.Response, error) bool {
		return true
	}
	cb := NewCircuitBreakerService(CircuitBreakerSettings{IsSuccessful: isSuccessful})

	for i := 0; i < 5; i++ {
		assert.Nil(t, fail(cb))
	}
	assert.Equal(t, CircuitBreakerStateClosed, cb.State())
	assert.Equal(t, CircuitBreakerCounts{5, 5, 0, 5, 0}, cb.counts)

	cb.counts.clear()

	cb.isSuccessful = func(_ *http.Response, err error) bool {
		return err == nil
	}
	for i := 0; i < 6; i++ {
		assert.Nil(t, fail(cb))
	}
	assert.Equal(t, CircuitBreakerStateOpen, cb.State())

}

func TestCircuitBreakerInParallel(t *testing.T) {
	customCB := newCustom()
	runtime.GOMAXPROCS(runtime.NumCPU())

	ch := make(chan error)

	const numReqs = 10000
	routine := func() {
		for i := 0; i < numReqs; i++ {
			ch <- succeed(customCB)
		}
	}

	const numRoutines = 10
	for i := 0; i < numRoutines; i++ {
		go routine()
	}

	total := uint32(numReqs * numRoutines)
	for i := uint32(0); i < total; i++ {
		err := <-ch
		assert.Nil(t, err)
	}
	assert.Equal(t, CircuitBreakerCounts{total, total, 0, total, 0}, customCB.counts)
}

func newNegativeDurationCB() *CircuitBreakerService {
	var negativeSt CircuitBreakerSettings

	negativeSt.Interval = time.Duration(-30) * time.Second
	negativeSt.Timeout = time.Duration(-90) * time.Second

	return NewCircuitBreakerService(negativeSt)
}

func newCustom() *CircuitBreakerService {
	var customSt CircuitBreakerSettings
	customSt.MaxRequests = 3
	customSt.Interval = time.Duration(30) * time.Second
	customSt.Timeout = time.Duration(90) * time.Second
	customSt.ReadyToTrip = func(counts CircuitBreakerCounts) bool {
		numReqs := counts.Requests
		failureRatio := float64(counts.TotalFailures) / float64(numReqs)

		counts.clear() // no effect on customCB.counts

		return numReqs >= 3 && failureRatio >= 0.6
	}
	customSt.OnStateChange = func(from CircuitBreakerState, to CircuitBreakerState) {
		stateChange = StateChange{from, to}
	}

	return NewCircuitBreakerService(customSt)
}
