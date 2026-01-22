package utils

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"biblio-audiobook-builder-tts/internal/logger"
)

// JobDispatcher manages a pool of workers to process jobs concurrently
type JobDispatcher struct {
	workers   []worker
	jobs      []job
	stopFlag  atomic.Bool
	mu        sync.RWMutex
	completed atomic.Int32
}

type worker struct {
	id   int
	busy atomic.Bool
	job  *job
}

type job struct {
	id       int
	jobFn    interface{}
	params   []interface{}
	assigned bool
	complete bool
}

// NewJobDispatcher creates a new job dispatcher with the specified number of workers
func NewJobDispatcher(numWorkers int) *JobDispatcher {
	if numWorkers < 1 {
		numWorkers = 1
	}
	jd := &JobDispatcher{
		workers: make([]worker, numWorkers),
		jobs:    make([]job, 0),
	}
	for i := range jd.workers {
		jd.workers[i].id = i
		jd.workers[i].busy.Store(false)
	}
	return jd
}

// AddJob adds a job to the dispatcher queue
// jobFn should be a function, params are the arguments to pass to it
func (d *JobDispatcher) AddJob(id int, jobFn interface{}, params ...interface{}) {
	t := reflect.TypeOf(jobFn)
	if t.Kind() != reflect.Func {
		logger.Error("JobDispatcher: AddJob requires a function, got %v", t.Kind())
		return
	}
	j := job{
		id:       id,
		jobFn:    jobFn,
		params:   params,
		complete: false,
	}
	d.mu.Lock()
	d.jobs = append(d.jobs, j)
	d.mu.Unlock()
}

// Start begins processing all queued jobs
// This function blocks until all jobs are complete or Stop is called
func (d *JobDispatcher) Start() {
	d.stopFlag.Store(false)
	d.completed.Store(0)

	// Assign jobs to workers
	d.mu.Lock()
	jobCount := len(d.jobs)
	d.mu.Unlock()

	for i := 0; i < jobCount; i++ {
		if d.stopFlag.Load() {
			break
		}

		d.mu.Lock()
		j := &d.jobs[i]
		d.mu.Unlock()

		if j.assigned {
			continue
		}

		// Find/wait for free worker
		var freeWorker *worker
		for freeWorker == nil && !d.stopFlag.Load() {
			for ii := range d.workers {
				w := &d.workers[ii]
				if !w.busy.Load() {
					freeWorker = w
					break
				}
			}
			if freeWorker == nil {
				time.Sleep(200 * time.Microsecond)
			}
		}

		if freeWorker == nil {
			break // Stopped
		}

		freeWorker.busy.Store(true)
		j.assigned = true
		go d.runJob(freeWorker, j)
	}

	// Wait for all jobs to complete
	for !d.stopFlag.Load() {
		allComplete := true
		d.mu.RLock()
		for i := range d.jobs {
			if !d.jobs[i].complete {
				allComplete = false
				break
			}
		}
		d.mu.RUnlock()

		if allComplete {
			break
		}
		time.Sleep(200 * time.Microsecond)
	}
}

// runJob executes a job on a worker
func (d *JobDispatcher) runJob(w *worker, j *job) {
	defer func() {
		d.mu.Lock()
		j.complete = true
		w.job = nil
		d.mu.Unlock()
		w.busy.Store(false)
		d.completed.Add(1)
	}()

	if d.stopFlag.Load() {
		return
	}

	d.mu.Lock()
	w.job = j
	d.mu.Unlock()

	f := reflect.ValueOf(j.jobFn)
	if len(j.params) != f.Type().NumIn() {
		logger.Error("JobDispatcher: Wrong number of parameters for job %d: got %d, want %d",
			j.id, len(j.params), f.Type().NumIn())
		return
	}

	in := make([]reflect.Value, len(j.params))
	for k, param := range j.params {
		in[k] = reflect.ValueOf(param)
	}
	f.Call(in)
}

// Stop signals all workers to stop processing
func (d *JobDispatcher) Stop() {
	d.stopFlag.Store(true)
}

// IsStopped returns true if the dispatcher has been stopped
func (d *JobDispatcher) IsStopped() bool {
	return d.stopFlag.Load()
}

// IsComplete returns true if the specified job is complete
func (d *JobDispatcher) IsComplete(jobId int) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for i := range d.jobs {
		if d.jobs[i].id == jobId {
			return d.jobs[i].complete
		}
	}
	return false
}

// GetProgress returns the number of completed jobs and total jobs
func (d *JobDispatcher) GetProgress() (completed int, total int) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	total = len(d.jobs)
	completed = int(d.completed.Load())
	return
}

// GetActiveWorkers returns information about currently active workers
func (d *JobDispatcher) GetActiveWorkers() []WorkerStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var active []WorkerStatus
	for i := range d.workers {
		w := &d.workers[i]
		if w.busy.Load() && w.job != nil {
			active = append(active, WorkerStatus{
				WorkerID: w.id,
				JobID:    w.job.id,
				Busy:     true,
			})
		}
	}
	return active
}

// WorkerStatus represents the current status of a worker
type WorkerStatus struct {
	WorkerID int
	JobID    int
	Busy     bool
}
