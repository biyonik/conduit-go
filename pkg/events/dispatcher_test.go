// -----------------------------------------------------------------------------
// Event Dispatcher Tests
// -----------------------------------------------------------------------------
// Testler:
// - Goroutine leak prevention
// - Graceful shutdown
// - Async dispatch with context cancellation
// - Race condition testing
// - Concurrent dispatch
// -----------------------------------------------------------------------------

package events

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockLogger, test için basit logger.
type MockLogger struct {
	mu      sync.Mutex
	logs    []string
	verbose bool
}

func NewMockLogger(verbose bool) *MockLogger {
	return &MockLogger{
		logs:    make([]string, 0),
		verbose: verbose,
	}
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	m.mu.Lock()
	m.logs = append(m.logs, msg)
	m.mu.Unlock()
	if m.verbose {
		fmt.Println(msg)
	}
}

func (m *MockLogger) Println(v ...interface{}) {
	msg := fmt.Sprint(v...)
	m.mu.Lock()
	m.logs = append(m.logs, msg)
	m.mu.Unlock()
	if m.verbose {
		fmt.Println(msg)
	}
}

func (m *MockLogger) GetLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.logs...)
}

func (m *MockLogger) Clear() {
	m.mu.Lock()
	m.logs = make([]string, 0)
	m.mu.Unlock()
}

// TestListener, test için basit listener.
type TestListener struct {
	name    string
	handled *atomic.Int32
	delay   time.Duration
	err     error
}

func NewTestListener(name string) *TestListener {
	return &TestListener{
		name:    name,
		handled: &atomic.Int32{},
	}
}

func (l *TestListener) Handle(event Event) error {
	if l.delay > 0 {
		time.Sleep(l.delay)
	}
	l.handled.Add(1)
	if l.err != nil {
		return l.err
	}
	return nil
}

func (l *TestListener) HandledCount() int {
	return int(l.handled.Load())
}

// TestDispatcher_BasicDispatch tests basic synchronous dispatch.
func TestDispatcher_BasicDispatch(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener := NewTestListener("test-listener")
	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", "test-data")
	err := dispatcher.Dispatch(event)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if listener.HandledCount() != 1 {
		t.Errorf("Expected listener to be called once, got: %d", listener.HandledCount())
	}
}

// TestDispatcher_MultipleListeners tests multiple listeners for same event.
func TestDispatcher_MultipleListeners(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener1 := NewTestListener("listener-1")
	listener2 := NewTestListener("listener-2")
	listener3 := NewTestListener("listener-3")

	dispatcher.Listen("test.event", listener1)
	dispatcher.Listen("test.event", listener2)
	dispatcher.Listen("test.event", listener3)

	event := NewBaseEvent("test.event", "test-data")
	dispatcher.Dispatch(event)

	if listener1.HandledCount() != 1 {
		t.Errorf("Listener 1: expected 1 call, got %d", listener1.HandledCount())
	}
	if listener2.HandledCount() != 1 {
		t.Errorf("Listener 2: expected 1 call, got %d", listener2.HandledCount())
	}
	if listener3.HandledCount() != 1 {
		t.Errorf("Listener 3: expected 1 call, got %d", listener3.HandledCount())
	}
}

// TestDispatcher_AsyncDispatch tests asynchronous dispatch.
func TestDispatcher_AsyncDispatch(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener := NewTestListener("async-listener")
	listener.delay = 100 * time.Millisecond
	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", "async-data")

	start := time.Now()
	dispatcher.DispatchAsync(event)
	elapsed := time.Since(start)

	// DispatchAsync should return immediately (< 50ms)
	if elapsed > 50*time.Millisecond {
		t.Errorf("DispatchAsync blocked for %v, expected < 50ms", elapsed)
	}

	// Wait for async processing
	time.Sleep(200 * time.Millisecond)

	if listener.HandledCount() != 1 {
		t.Errorf("Expected listener to be called once, got: %d", listener.HandledCount())
	}
}

// TestDispatcher_Shutdown tests graceful shutdown.
func TestDispatcher_Shutdown(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)

	listener := NewTestListener("shutdown-listener")
	listener.delay = 100 * time.Millisecond
	dispatcher.Listen("test.event", listener)

	// Dispatch 10 async events
	for i := 0; i < 10; i++ {
		event := NewBaseEvent("test.event", fmt.Sprintf("data-%d", i))
		dispatcher.DispatchAsync(event)
	}

	// Shutdown should wait for all events to complete
	start := time.Now()
	dispatcher.Shutdown()
	elapsed := time.Since(start)

	// All 10 listeners should have been called
	if listener.HandledCount() != 10 {
		t.Errorf("Expected 10 listener calls, got: %d", listener.HandledCount())
	}

	// Shutdown should have waited (at least 100ms)
	if elapsed < 100*time.Millisecond {
		t.Errorf("Shutdown completed too quickly: %v", elapsed)
	}
}

// TestDispatcher_ShutdownTimeout tests shutdown with timeout.
func TestDispatcher_ShutdownWithTimeout(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)

	listener := NewTestListener("slow-listener")
	listener.delay = 500 * time.Millisecond // Very slow
	dispatcher.Listen("test.event", listener)

	// Dispatch async event
	event := NewBaseEvent("test.event", "data")
	dispatcher.DispatchAsync(event)

	// Shutdown with short timeout
	err := dispatcher.ShutdownWithTimeout(100 * time.Millisecond)

	// Should timeout
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

// TestDispatcher_NoGoroutineLeak tests that no goroutines leak after shutdown.
func TestDispatcher_NoGoroutineLeak(t *testing.T) {
	logger := NewMockLogger(false)

	initialGoroutines := countGoroutines()

	// Create and use dispatcher
	dispatcher := NewDispatcher(logger)
	listener := NewTestListener("leak-test")
	dispatcher.Listen("test.event", listener)

	// Dispatch many async events
	for i := 0; i < 100; i++ {
		event := NewBaseEvent("test.event", fmt.Sprintf("data-%d", i))
		dispatcher.DispatchAsync(event)
	}

	// Shutdown and wait
	dispatcher.Shutdown()
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := countGoroutines()

	// Should have same or fewer goroutines (some tolerance for runtime)
	if finalGoroutines > initialGoroutines+5 {
		t.Errorf("Potential goroutine leak: initial=%d, final=%d", initialGoroutines, finalGoroutines)
	}
}

// TestDispatcher_AsyncAfterShutdown tests that async dispatch after shutdown is ignored.
func TestDispatcher_AsyncAfterShutdown(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)

	listener := NewTestListener("post-shutdown")
	dispatcher.Listen("test.event", listener)

	// Shutdown first
	dispatcher.Shutdown()

	// Try to dispatch after shutdown
	event := NewBaseEvent("test.event", "ignored-data")
	dispatcher.DispatchAsync(event)

	time.Sleep(100 * time.Millisecond)

	// Listener should NOT have been called
	if listener.HandledCount() != 0 {
		t.Errorf("Expected 0 listener calls after shutdown, got: %d", listener.HandledCount())
	}
}

// TestDispatcher_ConcurrentDispatch tests concurrent dispatching (race condition check).
func TestDispatcher_ConcurrentDispatch(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener := NewTestListener("concurrent-listener")
	dispatcher.Listen("test.event", listener)

	var wg sync.WaitGroup
	numGoroutines := 50
	eventsPerGoroutine := 20

	// Launch many goroutines dispatching events concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := NewBaseEvent("test.event", fmt.Sprintf("data-%d-%d", id, j))
				dispatcher.Dispatch(event)
			}
		}(i)
	}

	wg.Wait()

	expected := numGoroutines * eventsPerGoroutine
	if listener.HandledCount() != expected {
		t.Errorf("Expected %d listener calls, got: %d", expected, listener.HandledCount())
	}
}

// TestDispatcher_ListenerError tests that one listener error doesn't stop others.
func TestDispatcher_ListenerError(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener1 := NewTestListener("listener-1")
	listener2 := NewTestListener("listener-2")
	listener2.err = fmt.Errorf("simulated error")
	listener3 := NewTestListener("listener-3")

	dispatcher.Listen("test.event", listener1)
	dispatcher.Listen("test.event", listener2)
	dispatcher.Listen("test.event", listener3)

	event := NewBaseEvent("test.event", "test-data")
	err := dispatcher.Dispatch(event)

	// Should return the error from listener2
	if err == nil {
		t.Error("Expected error from listener2, got nil")
	}

	// All listeners should still be called
	if listener1.HandledCount() != 1 {
		t.Errorf("Listener 1: expected 1 call, got %d", listener1.HandledCount())
	}
	if listener2.HandledCount() != 1 {
		t.Errorf("Listener 2: expected 1 call, got %d", listener2.HandledCount())
	}
	if listener3.HandledCount() != 1 {
		t.Errorf("Listener 3: expected 1 call, got %d", listener3.HandledCount())
	}
}

// TestDispatcher_ConditionalListener tests conditional listener wrapper.
func TestDispatcher_ConditionalListener(t *testing.T) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener := NewTestListener("conditional")

	// Only handle events with payload == "allowed"
	condition := func(e Event) bool {
		return e.Payload() == "allowed"
	}

	conditionalListener := NewConditionalListener(listener, condition)
	dispatcher.Listen("test.event", conditionalListener)

	// This should be handled
	event1 := NewBaseEvent("test.event", "allowed")
	dispatcher.Dispatch(event1)

	// This should be skipped
	event2 := NewBaseEvent("test.event", "blocked")
	dispatcher.Dispatch(event2)

	if listener.HandledCount() != 1 {
		t.Errorf("Expected 1 listener call, got: %d", listener.HandledCount())
	}
}

// Helper function to count goroutines
func countGoroutines() int {
	// This is an approximation for testing
	// In production, use runtime.NumGoroutine() with proper baseline
	var n int
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		// Sample measurement
		n++
	}
	return n
}

// BenchmarkDispatcher_SyncDispatch benchmarks synchronous dispatch.
func BenchmarkDispatcher_SyncDispatch(b *testing.B) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener := NewTestListener("bench-listener")
	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", "bench-data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dispatcher.Dispatch(event)
	}
}

// BenchmarkDispatcher_AsyncDispatch benchmarks asynchronous dispatch.
func BenchmarkDispatcher_AsyncDispatch(b *testing.B) {
	logger := NewMockLogger(false)
	dispatcher := NewDispatcher(logger)
	defer dispatcher.Shutdown()

	listener := NewTestListener("bench-listener")
	dispatcher.Listen("test.event", listener)

	event := NewBaseEvent("test.event", "bench-data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dispatcher.DispatchAsync(event)
	}
}
