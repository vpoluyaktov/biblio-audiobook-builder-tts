package server

import (
	"sync"
	"time"

	"abb_tts/internal/parser"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a conversion job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusParsing    JobStatus = "parsing"
	JobStatusConverting JobStatus = "converting"
	JobStatusBuilding   JobStatus = "building"  // Building M4B file
	JobStatusUploading  JobStatus = "uploading" // Uploading to Audiobookshelf
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// WorkerProgress represents the progress of a single parallel worker
type WorkerProgress struct {
	WorkerID       int     `json:"worker_id"`
	ChapterIndex   int     `json:"chapter_index"`
	ChapterTitle   string  `json:"chapter_title"`
	Progress       float64 `json:"progress"` // 0.0 to 1.0
	ChunksTotal    int     `json:"chunks_total"`
	ChunksComplete int     `json:"chunks_complete"`
	Active         bool    `json:"active"`
}

// FailedChapter represents a chapter that failed to convert
type FailedChapter struct {
	Index int    `json:"index"`
	Title string `json:"title"`
	Error string `json:"error"`
}

// JobDTO is a data transfer object for Job without mutex (safe for JSON serialization)
type JobDTO struct {
	ID                string           `json:"id"`
	FileName          string           `json:"file_name"`
	Status            JobStatus        `json:"status"`
	Progress          float64          `json:"progress"`
	CurrentChapter    string           `json:"current_chapter"`
	TotalChapters     int              `json:"total_chapters"`
	CurrentChapterNum int              `json:"current_chapter_num"`
	WorkerProgress    []WorkerProgress `json:"worker_progress,omitempty"`
	NumWorkers        int              `json:"num_workers,omitempty"`
	Provider          string           `json:"provider"`
	Voice             string           `json:"voice"`
	Speed             float64          `json:"speed"`
	Pitch             float64          `json:"pitch"`
	BookTitle         string           `json:"book_title"`
	BookAuthor        string           `json:"book_author"`
	OutputPath        string           `json:"output_path,omitempty"`
	M4BFile           string           `json:"m4b_file,omitempty"`
	M4BFiles          []string         `json:"m4b_files,omitempty"`
	FailedChapters    []FailedChapter  `json:"failed_chapters,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	StartedAt         *time.Time       `json:"started_at,omitempty"`
	CompletedAt       *time.Time       `json:"completed_at,omitempty"`
	Error             string           `json:"error,omitempty"`
}

// Job represents a book-to-audiobook conversion job
type Job struct {
	ID                string    `json:"id"`
	FileName          string    `json:"file_name"`
	FilePath          string    `json:"-"` // Internal path, not exposed to API
	Status            JobStatus `json:"status"`
	Progress          float64   `json:"progress"`        // 0.0 to 1.0
	CurrentChapter    string    `json:"current_chapter"` // Currently processing chapter
	TotalChapters     int       `json:"total_chapters"`
	CurrentChapterNum int       `json:"current_chapter_num"`

	// Parallel processing progress
	WorkerProgress []WorkerProgress `json:"worker_progress,omitempty"`
	NumWorkers     int              `json:"num_workers,omitempty"`

	// TTS settings
	Provider string  `json:"provider"`
	Voice    string  `json:"voice"`
	Speed    float64 `json:"speed"`
	Pitch    float64 `json:"pitch"`

	// Book metadata (populated after parsing)
	BookTitle  string `json:"book_title"`
	BookAuthor string `json:"book_author"`

	// Output
	OutputPath     string          `json:"output_path,omitempty"`
	M4BFile        string          `json:"m4b_file,omitempty"`  // Primary M4B file (or first part)
	M4BFiles       []string        `json:"m4b_files,omitempty"` // All M4B files (for multi-part)
	ChapterFiles   []string        `json:"-"`                   // Internal list of chapter audio files
	FailedChapters []FailedChapter `json:"failed_chapters,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error info
	Error string `json:"error,omitempty"`

	// Internal fields (not serialized)
	book *parser.Book `json:"-"`
	mu   sync.RWMutex `json:"-"`
}

// NewJob creates a new conversion job
func NewJob(fileName, filePath, provider, voice string, speed, pitch float64) *Job {
	return &Job{
		ID:        uuid.New().String(),
		FileName:  fileName,
		FilePath:  filePath,
		Status:    JobStatusPending,
		Progress:  0,
		Provider:  provider,
		Voice:     voice,
		Speed:     speed,
		Pitch:     pitch,
		CreatedAt: time.Now(),
	}
}

// SetStatus updates the job status thread-safely
func (j *Job) SetStatus(status JobStatus) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status

	if status == JobStatusConverting && j.StartedAt == nil {
		now := time.Now()
		j.StartedAt = &now
	}

	if status == JobStatusCompleted || status == JobStatusFailed || status == JobStatusCancelled {
		now := time.Now()
		j.CompletedAt = &now
	}
}

// SetProgress updates the job progress thread-safely
func (j *Job) SetProgress(progress float64, currentChapter string, chapterNum int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Progress = progress
	j.CurrentChapter = currentChapter
	j.CurrentChapterNum = chapterNum
}

// InitWorkerProgress initializes the worker progress array
func (j *Job) InitWorkerProgress(numWorkers int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.NumWorkers = numWorkers
	j.WorkerProgress = make([]WorkerProgress, numWorkers)
	for i := 0; i < numWorkers; i++ {
		j.WorkerProgress[i] = WorkerProgress{
			WorkerID: i,
			Active:   false,
		}
	}
}

// SetWorkerProgress updates a specific worker's progress
func (j *Job) SetWorkerProgress(workerID int, chapterIndex int, chapterTitle string, chunksComplete, chunksTotal int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if workerID >= 0 && workerID < len(j.WorkerProgress) {
		progress := 0.0
		if chunksTotal > 0 {
			progress = float64(chunksComplete) / float64(chunksTotal)
		}
		j.WorkerProgress[workerID] = WorkerProgress{
			WorkerID:       workerID,
			ChapterIndex:   chapterIndex,
			ChapterTitle:   chapterTitle,
			Progress:       progress,
			ChunksTotal:    chunksTotal,
			ChunksComplete: chunksComplete,
			Active:         true,
		}
	}
}

// ClearWorkerProgress marks a worker as inactive (finished its chapter)
func (j *Job) ClearWorkerProgress(workerID int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if workerID >= 0 && workerID < len(j.WorkerProgress) {
		j.WorkerProgress[workerID].Active = false
		j.WorkerProgress[workerID].Progress = 1.0
	}
}

// SetBook sets the parsed book data
func (j *Job) SetBook(book *parser.Book) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.book = book
	if book != nil {
		j.BookTitle = book.Title
		j.BookAuthor = book.Author
		j.TotalChapters = len(book.Chapters)
	}
}

// GetBook returns the parsed book data
func (j *Job) GetBook() *parser.Book {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.book
}

// SetError sets an error message and marks the job as failed
func (j *Job) SetError(err string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Error = err
	j.Status = JobStatusFailed
	now := time.Now()
	j.CompletedAt = &now
}

// AddFailedChapter records a chapter that failed to convert
func (j *Job) AddFailedChapter(index int, title string, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.FailedChapters = append(j.FailedChapters, FailedChapter{
		Index: index,
		Title: title,
		Error: err.Error(),
	})
}

// SetOutputPath sets the output path for the completed audiobook
func (j *Job) SetOutputPath(path string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.OutputPath = path
}

// Clone returns a copy of the job safe for JSON serialization
func (j *Job) Clone() JobDTO {
	j.mu.RLock()
	defer j.mu.RUnlock()

	// Create a JobDTO with copied fields (no mutex)
	clone := JobDTO{
		ID:                j.ID,
		FileName:          j.FileName,
		Status:            j.Status,
		Progress:          j.Progress,
		CurrentChapter:    j.CurrentChapter,
		TotalChapters:     j.TotalChapters,
		CurrentChapterNum: j.CurrentChapterNum,
		NumWorkers:        j.NumWorkers,
		Provider:          j.Provider,
		Voice:             j.Voice,
		Speed:             j.Speed,
		Pitch:             j.Pitch,
		BookTitle:         j.BookTitle,
		BookAuthor:        j.BookAuthor,
		OutputPath:        j.OutputPath,
		M4BFile:           j.M4BFile,
		CreatedAt:         j.CreatedAt,
		StartedAt:         j.StartedAt,
		CompletedAt:       j.CompletedAt,
		Error:             j.Error,
	}

	// Copy slices
	if j.WorkerProgress != nil {
		clone.WorkerProgress = make([]WorkerProgress, len(j.WorkerProgress))
		copy(clone.WorkerProgress, j.WorkerProgress)
	}
	if j.M4BFiles != nil {
		clone.M4BFiles = make([]string, len(j.M4BFiles))
		copy(clone.M4BFiles, j.M4BFiles)
	}
	if j.FailedChapters != nil {
		clone.FailedChapters = make([]FailedChapter, len(j.FailedChapters))
		copy(clone.FailedChapters, j.FailedChapters)
	}

	return clone
}

// JobStore manages all jobs in memory
type JobStore struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewJobStore creates a new job store
func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*Job),
	}
}

// Add adds a job to the store
func (s *JobStore) Add(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Get retrieves a job by ID
func (s *JobStore) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.jobs[id]
	return job, exists
}

// Delete removes a job from the store
func (s *JobStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[id]; exists {
		delete(s.jobs, id)
		return true
	}
	return false
}

// List returns all jobs (cloned for safe serialization)
func (s *JobStore) List() []JobDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]JobDTO, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job.Clone())
	}
	return jobs
}

// GetPending returns the next pending job (FIFO by creation time)
func (s *JobStore) GetPending() *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var oldest *Job
	for _, job := range s.jobs {
		if job.Status == JobStatusPending {
			if oldest == nil || job.CreatedAt.Before(oldest.CreatedAt) {
				oldest = job
			}
		}
	}
	return oldest
}

// Count returns the total number of jobs
func (s *JobStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs)
}

// CountByStatus returns the number of jobs with a specific status
func (s *JobStore) CountByStatus(status JobStatus) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, job := range s.jobs {
		if job.Status == status {
			count++
		}
	}
	return count
}
