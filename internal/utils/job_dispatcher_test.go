package utils

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewJobDispatcher(t *testing.T) {
	jd := NewJobDispatcher(3)
	if len(jd.workers) != 3 {
		t.Errorf("Expected 3 workers, got %d", len(jd.workers))
	}

	// Test minimum workers
	jd = NewJobDispatcher(0)
	if len(jd.workers) != 1 {
		t.Errorf("Expected minimum 1 worker, got %d", len(jd.workers))
	}
}

func TestJobDispatcherAddJob(t *testing.T) {
	jd := NewJobDispatcher(2)

	testFn := func(val int) {
		// Function body for testing
	}

	jd.AddJob(1, testFn, 42)

	if len(jd.jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jd.jobs))
	}
	if jd.jobs[0].id != 1 {
		t.Errorf("Expected job id 1, got %d", jd.jobs[0].id)
	}
}

func TestJobDispatcherStart(t *testing.T) {
	jd := NewJobDispatcher(2)

	var results sync.Map
	testFn := func(id int) {
		time.Sleep(10 * time.Millisecond)
		results.Store(id, true)
	}

	// Add 5 jobs
	for i := 0; i < 5; i++ {
		jd.AddJob(i, testFn, i)
	}

	jd.Start()

	// Verify all jobs completed
	for i := 0; i < 5; i++ {
		if _, ok := results.Load(i); !ok {
			t.Errorf("Job %d did not complete", i)
		}
	}
}

func TestJobDispatcherParallelExecution(t *testing.T) {
	jd := NewJobDispatcher(3)

	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	testFn := func(id int) {
		current := currentConcurrent.Add(1)
		// Track max concurrent
		for {
			max := maxConcurrent.Load()
			if current <= max || maxConcurrent.CompareAndSwap(max, current) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		currentConcurrent.Add(-1)
	}

	// Add 6 jobs
	for i := 0; i < 6; i++ {
		jd.AddJob(i, testFn, i)
	}

	jd.Start()

	// With 3 workers and 6 jobs, we should see at least 2 concurrent executions
	if maxConcurrent.Load() < 2 {
		t.Errorf("Expected at least 2 concurrent executions, got %d", maxConcurrent.Load())
	}
}

func TestJobDispatcherGetProgress(t *testing.T) {
	jd := NewJobDispatcher(2)

	testFn := func(id int) {
		time.Sleep(10 * time.Millisecond)
	}

	for i := 0; i < 4; i++ {
		jd.AddJob(i, testFn, i)
	}

	completed, total := jd.GetProgress()
	if total != 4 {
		t.Errorf("Expected total 4, got %d", total)
	}
	if completed != 0 {
		t.Errorf("Expected completed 0 before start, got %d", completed)
	}

	jd.Start()

	completed, total = jd.GetProgress()
	if completed != 4 {
		t.Errorf("Expected completed 4 after start, got %d", completed)
	}
}

func TestJobDispatcherStop(t *testing.T) {
	jd := NewJobDispatcher(1)

	var completedJobs atomic.Int32
	testFn := func(id int) {
		time.Sleep(100 * time.Millisecond)
		completedJobs.Add(1)
	}

	// Add many jobs
	for i := 0; i < 10; i++ {
		jd.AddJob(i, testFn, i)
	}

	// Start in goroutine and stop after short delay
	go func() {
		time.Sleep(150 * time.Millisecond)
		jd.Stop()
	}()

	jd.Start()

	// Should have completed only 1-2 jobs before stop
	if completedJobs.Load() >= 5 {
		t.Errorf("Expected fewer than 5 completed jobs after stop, got %d", completedJobs.Load())
	}
}

func TestJobDispatcherIsComplete(t *testing.T) {
	jd := NewJobDispatcher(2)

	testFn := func(id int) {
		time.Sleep(10 * time.Millisecond)
	}

	jd.AddJob(1, testFn, 1)
	jd.AddJob(2, testFn, 2)

	if jd.IsComplete(1) {
		t.Error("Job 1 should not be complete before start")
	}

	jd.Start()

	if !jd.IsComplete(1) {
		t.Error("Job 1 should be complete after start")
	}
	if !jd.IsComplete(2) {
		t.Error("Job 2 should be complete after start")
	}
}

func TestJobDispatcherWithDifferentParamTypes(t *testing.T) {
	jd := NewJobDispatcher(2)

	var result string
	var mu sync.Mutex

	testFn := func(name string, count int, flag bool) {
		mu.Lock()
		if flag {
			result = name
		}
		mu.Unlock()
	}

	jd.AddJob(1, testFn, "test", 42, true)
	jd.Start()

	mu.Lock()
	if result != "test" {
		t.Errorf("Expected result 'test', got '%s'", result)
	}
	mu.Unlock()
}

func TestJobDispatcherSingleWorker(t *testing.T) {
	// Test that single worker processes jobs sequentially
	jd := NewJobDispatcher(1)

	var order []int
	var mu sync.Mutex

	testFn := func(id int) {
		mu.Lock()
		order = append(order, id)
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}

	for i := 0; i < 5; i++ {
		jd.AddJob(i, testFn, i)
	}

	jd.Start()

	mu.Lock()
	defer mu.Unlock()

	if len(order) != 5 {
		t.Errorf("Expected 5 jobs completed, got %d", len(order))
	}

	// With single worker, jobs should complete in order
	for i := 0; i < 5; i++ {
		if order[i] != i {
			t.Errorf("Expected job %d at position %d, got %d", i, i, order[i])
		}
	}
}

func TestJobDispatcherManyWorkersFewJobs(t *testing.T) {
	// More workers than jobs
	jd := NewJobDispatcher(10)

	var completed atomic.Int32
	testFn := func(id int) {
		time.Sleep(10 * time.Millisecond)
		completed.Add(1)
	}

	// Only 3 jobs for 10 workers
	for i := 0; i < 3; i++ {
		jd.AddJob(i, testFn, i)
	}

	jd.Start()

	if completed.Load() != 3 {
		t.Errorf("Expected 3 completed jobs, got %d", completed.Load())
	}
}

func TestJobDispatcherEmptyJobs(t *testing.T) {
	jd := NewJobDispatcher(3)

	// Start with no jobs - should complete immediately
	jd.Start()

	completed, total := jd.GetProgress()
	if total != 0 {
		t.Errorf("Expected 0 total jobs, got %d", total)
	}
	if completed != 0 {
		t.Errorf("Expected 0 completed jobs, got %d", completed)
	}
}

func TestJobDispatcherAddNonFunction(t *testing.T) {
	jd := NewJobDispatcher(2)

	// Try to add a non-function - should be ignored
	jd.AddJob(1, "not a function", 42)

	if len(jd.jobs) != 0 {
		t.Errorf("Expected 0 jobs after adding non-function, got %d", len(jd.jobs))
	}
}

func TestJobDispatcherWrongParamCount(t *testing.T) {
	jd := NewJobDispatcher(2)

	var called atomic.Bool
	testFn := func(a, b int) {
		called.Store(true)
	}

	// Add job with wrong number of params (1 instead of 2)
	jd.AddJob(1, testFn, 42)
	jd.Start()

	// Function should not be called due to param mismatch
	if called.Load() {
		t.Error("Function should not have been called with wrong param count")
	}
}

func TestJobDispatcherConcurrentSafety(t *testing.T) {
	// Test for race conditions with concurrent access
	jd := NewJobDispatcher(5)

	var counter atomic.Int64
	testFn := func(id int) {
		counter.Add(1)
		time.Sleep(time.Duration(id) * time.Millisecond)
	}

	// Add 100 jobs
	for i := 0; i < 100; i++ {
		jd.AddJob(i, testFn, i%10)
	}

	// Start processing
	jd.Start()

	if counter.Load() != 100 {
		t.Errorf("Expected 100 jobs completed, got %d", counter.Load())
	}

	completed, total := jd.GetProgress()
	if completed != 100 || total != 100 {
		t.Errorf("Expected progress 100/100, got %d/%d", completed, total)
	}
}

func TestJobDispatcherGetActiveWorkers(t *testing.T) {
	jd := NewJobDispatcher(3)

	var activeChecked atomic.Bool
	var activeCount int

	testFn := func(id int) {
		if id == 0 {
			// First job checks active workers while others are running
			time.Sleep(20 * time.Millisecond)
			active := jd.GetActiveWorkers()
			activeCount = len(active)
			activeChecked.Store(true)
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}

	for i := 0; i < 3; i++ {
		jd.AddJob(i, testFn, i)
	}

	jd.Start()

	if !activeChecked.Load() {
		t.Error("Active workers check was not performed")
	}

	// Should have seen at least 2 active workers (including self)
	if activeCount < 2 {
		t.Errorf("Expected at least 2 active workers, got %d", activeCount)
	}
}

func TestJobDispatcherIsStopped(t *testing.T) {
	jd := NewJobDispatcher(2)

	if jd.IsStopped() {
		t.Error("Dispatcher should not be stopped initially")
	}

	jd.Stop()

	if !jd.IsStopped() {
		t.Error("Dispatcher should be stopped after Stop()")
	}
}

func TestJobDispatcherResultsOrder(t *testing.T) {
	// Verify that results maintain order even with parallel execution
	jd := NewJobDispatcher(4)

	results := make([]int, 20)
	var mu sync.Mutex

	testFn := func(id int, sleepMs int) {
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
		mu.Lock()
		results[id] = id * 2 // Store computed result
		mu.Unlock()
	}

	// Add jobs with varying sleep times to cause out-of-order completion
	for i := 0; i < 20; i++ {
		sleepMs := (20 - i) % 10 // Varying sleep times
		jd.AddJob(i, testFn, i, sleepMs)
	}

	jd.Start()

	// Verify all results are correct regardless of completion order
	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < 20; i++ {
		expected := i * 2
		if results[i] != expected {
			t.Errorf("Result[%d] = %d, expected %d", i, results[i], expected)
		}
	}
}

func TestJobDispatcherStopDuringExecution(t *testing.T) {
	jd := NewJobDispatcher(2)

	var startedJobs atomic.Int32
	var completedJobs atomic.Int32

	testFn := func(id int) {
		startedJobs.Add(1)
		time.Sleep(100 * time.Millisecond)
		completedJobs.Add(1)
	}

	// Add many jobs
	for i := 0; i < 20; i++ {
		jd.AddJob(i, testFn, i)
	}

	// Stop after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		jd.Stop()
	}()

	jd.Start()

	// Some jobs should have started but not all completed
	started := startedJobs.Load()
	completed := completedJobs.Load()

	if started == 0 {
		t.Error("Expected some jobs to have started")
	}
	if completed >= 10 {
		t.Errorf("Expected fewer than 10 completed jobs after stop, got %d", completed)
	}
}

func TestJobDispatcherNegativeWorkers(t *testing.T) {
	jd := NewJobDispatcher(-5)

	// Should default to 1 worker
	if len(jd.workers) != 1 {
		t.Errorf("Expected 1 worker for negative input, got %d", len(jd.workers))
	}
}

func TestJobDispatcherLargeNumberOfJobs(t *testing.T) {
	jd := NewJobDispatcher(10)

	var counter atomic.Int64
	testFn := func(id int) {
		counter.Add(1)
	}

	// Add 1000 jobs
	numJobs := 1000
	for i := 0; i < numJobs; i++ {
		jd.AddJob(i, testFn, i)
	}

	jd.Start()

	if counter.Load() != int64(numJobs) {
		t.Errorf("Expected %d jobs completed, got %d", numJobs, counter.Load())
	}
}

func TestJobDispatcherJobWithPanic(t *testing.T) {
	// Note: This test documents current behavior - panics in jobs will crash
	// In production, you may want to add panic recovery
	t.Skip("Skipping panic test - current implementation doesn't recover from panics")
}

func TestJobDispatcherIsCompleteNonExistent(t *testing.T) {
	jd := NewJobDispatcher(2)

	testFn := func(id int) {}
	jd.AddJob(1, testFn, 1)
	jd.Start()

	// Check for non-existent job ID
	if jd.IsComplete(999) {
		t.Error("Non-existent job should not be marked as complete")
	}
}
