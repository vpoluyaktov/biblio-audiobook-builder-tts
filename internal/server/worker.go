package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"abb_tts/internal/config"
	"abb_tts/internal/parser"
	"abb_tts/internal/tts"
)

// Worker processes conversion jobs from the queue
type Worker struct {
	store      *JobStore
	hub        *Hub
	ttsService tts.Service
	cfg        *config.Config
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewWorker creates a new job worker
func NewWorker(store *JobStore, hub *Hub, ttsService tts.Service, cfg *config.Config) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		store:      store,
		hub:        hub,
		ttsService: ttsService,
		cfg:        cfg,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins processing jobs
func (w *Worker) Start() {
	w.wg.Add(1)
	go w.processLoop()
	log.Println("Job worker started")
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.cancel()
	w.wg.Wait()
	log.Println("Job worker stopped")
}

// processLoop continuously checks for and processes pending jobs
func (w *Worker) processLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// Check for pending jobs
			job := w.store.GetPending()
			if job != nil {
				w.processJob(job)
			}
		}
	}
}

// processJob handles a single conversion job
func (w *Worker) processJob(job *Job) {
	log.Printf("Processing job %s: %s", job.ID, job.FileName)

	// Parse the book
	job.SetStatus(JobStatusParsing)
	w.broadcastJobUpdate(job)

	book, err := w.parseBook(job.FilePath)
	if err != nil {
		job.SetError(fmt.Sprintf("Failed to parse book: %v", err))
		w.broadcastJobFailed(job)
		return
	}

	job.SetBook(book)
	w.broadcastJobUpdate(job)

	// Start conversion
	job.SetStatus(JobStatusConverting)
	w.broadcastJobUpdate(job)

	outputPath, err := w.convertBook(job, book)
	if err != nil {
		job.SetError(fmt.Sprintf("Conversion failed: %v", err))
		w.broadcastJobFailed(job)
		return
	}

	// Mark as completed
	job.SetOutputPath(outputPath)
	job.SetStatus(JobStatusCompleted)
	job.SetProgress(1.0, "", len(book.Chapters))
	w.broadcastJobCompleted(job)

	log.Printf("Job %s completed: %s", job.ID, outputPath)
}

// parseBook parses the ebook file
func (w *Worker) parseBook(filePath string) (*parser.Book, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".epub":
		return parser.NewEpubParser().ParseEpubFile(filePath)
	case ".fb2":
		return parser.NewFB2Parser().ParseFB2File(filePath)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

// convertBook converts the book to audio files
func (w *Worker) convertBook(job *Job, book *parser.Book) (string, error) {
	// Create output directory
	outputDir := filepath.Join(w.cfg.OutputDir, sanitizeFileName(book.Title))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	totalChapters := len(book.Chapters)

	for i, chapter := range book.Chapters {
		// Check for cancellation
		select {
		case <-w.ctx.Done():
			return "", fmt.Errorf("conversion cancelled")
		default:
		}

		// Update progress
		progress := float64(i) / float64(totalChapters)
		job.SetProgress(progress, chapter.Title, i+1)
		w.broadcastJobProgress(job)

		// Convert chapter
		reader, err := w.ttsService.ConvertToSpeech(chapter.Content, &tts.ConversionOptions{
			Voice:    job.Voice,
			Provider: job.Provider,
			Speed:    job.Speed,
			Pitch:    job.Pitch,
		})
		if err != nil {
			log.Printf("Warning: Failed to convert chapter '%s': %v", chapter.Title, err)
			continue // Skip failed chapters but continue with others
		}

		// Save audio file
		chapterFileName := fmt.Sprintf("%02d_%s.mp3", i+1, sanitizeFileName(chapter.Title))
		outputPath := filepath.Join(outputDir, chapterFileName)

		outputFile, err := os.Create(outputPath)
		if err != nil {
			log.Printf("Warning: Failed to create output file for chapter '%s': %v", chapter.Title, err)
			continue
		}

		if _, err := outputFile.ReadFrom(reader); err != nil {
			outputFile.Close()
			log.Printf("Warning: Failed to write audio for chapter '%s': %v", chapter.Title, err)
			continue
		}
		outputFile.Close()

		log.Printf("Converted chapter %d/%d: %s", i+1, totalChapters, chapter.Title)
	}

	return outputDir, nil
}

// sanitizeFileName removes or replaces characters that are invalid in file names
func sanitizeFileName(name string) string {
	// Replace common problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	result := replacer.Replace(name)

	// Trim spaces and dots from ends
	result = strings.TrimSpace(result)
	result = strings.Trim(result, ".")

	// Limit length
	if len(result) > 100 {
		result = result[:100]
	}

	if result == "" {
		result = "untitled"
	}

	return result
}

// Broadcast helpers
func (w *Worker) broadcastJobUpdate(job *Job) {
	w.hub.Broadcast(WSMessage{
		Type:    WSTypeJobUpdated,
		Payload: job.Clone(),
	})
}

func (w *Worker) broadcastJobProgress(job *Job) {
	w.hub.Broadcast(WSMessage{
		Type:    WSTypeJobProgress,
		Payload: job.Clone(),
	})
}

func (w *Worker) broadcastJobCompleted(job *Job) {
	w.hub.Broadcast(WSMessage{
		Type:    WSTypeJobCompleted,
		Payload: job.Clone(),
	})
}

func (w *Worker) broadcastJobFailed(job *Job) {
	w.hub.Broadcast(WSMessage{
		Type:    WSTypeJobFailed,
		Payload: job.Clone(),
	})
}
