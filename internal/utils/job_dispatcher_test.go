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
