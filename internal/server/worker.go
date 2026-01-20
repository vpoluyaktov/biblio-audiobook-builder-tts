package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"abb_tts/internal/audio"
	"abb_tts/internal/audiobookshelf"
	"abb_tts/internal/config"
	"abb_tts/internal/logger"
	"abb_tts/internal/normalize"
	"abb_tts/internal/parser"
	"abb_tts/internal/sanitize"
	"abb_tts/internal/storage"
	"abb_tts/internal/tts"
	"abb_tts/internal/utils"
)

// JobDB defines the database operations needed for job management
type JobDB interface {
	GetJob(id string) (*storage.Job, error)
	GetPendingJob() (*storage.Job, error)
	CreateJob(job *storage.Job) error
	UpdateJob(job *storage.Job) error
	DeleteJob(id string) error
}

// Worker processes conversion jobs from the queue
type Worker struct {
	db         JobDB
	hub        *Hub
	ttsService tts.Service
	cfg        *config.Config
	sanitizer  *sanitize.TextSanitizer
	normalizer *normalize.Processor
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewWorker creates a new job worker
func NewWorker(db JobDB, hub *Hub, ttsService tts.Service, cfg *config.Config) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize text sanitizer with pronunciation dictionary
	textSanitizer := sanitize.NewTextSanitizer()

	// Load default rules if enabled
	if cfg.UseDefaultPronunciation {
		textSanitizer.LoadDefaultRules()
		logger.Info("Loaded %d default pronunciation rules", textSanitizer.GetDictionary().RuleCount())
	}

	// Load custom dictionary if specified
	if cfg.PronunciationDictFile != "" {
		if err := textSanitizer.GetDictionary().LoadFromFile(cfg.PronunciationDictFile); err != nil {
			logger.Warn("Failed to load pronunciation dictionary: %v", err)
		} else {
			logger.Info("Loaded pronunciation dictionary from %s", cfg.PronunciationDictFile)
		}
	}

	return &Worker{
		db:         db,
		hub:        hub,
		ttsService: ttsService,
		cfg:        cfg,
		sanitizer:  textSanitizer,
		normalizer: normalize.NewProcessor(),
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
			// Check for pending jobs from database
			dbJob, err := w.db.GetPendingJob()
			if err != nil {
				logger.Warn("Failed to get pending job: %v", err)
				continue
			}
			if dbJob != nil {
				job := storageJobToJob(dbJob)
				w.processJob(job)
			}
		}
	}
}

// saveJob persists job state to the database
func (w *Worker) saveJob(job *Job) {
	if w.db != nil {
		dbJob := jobToStorageJob(job)
		if err := w.db.UpdateJob(dbJob); err != nil {
			logger.Warn("Failed to save job to database: %v", err)
		}
	}
}

// processJob handles a single conversion job
func (w *Worker) processJob(job *Job) {
	logger.Info("Processing job %s: %s", job.ID, job.FileName)

	// Parse the book
	job.SetStatus(JobStatusParsing)
	w.saveJob(job)
	w.broadcastJobUpdate(job)

	book, err := w.parseBook(job.FilePath)
	if err != nil {
		job.SetError(fmt.Sprintf("Failed to parse book: %v", err))
		w.saveJob(job)
		w.broadcastJobFailed(job)
		return
	}

	job.SetBook(book)
	w.saveJob(job)
	w.broadcastJobUpdate(job)

	// Start conversion
	job.SetStatus(JobStatusConverting)
	w.saveJob(job)
	w.broadcastJobUpdate(job)

	outputDir, chapterFiles, err := w.convertBook(job, book)
	if err != nil {
		job.SetError(fmt.Sprintf("Conversion failed: %v", err))
		w.saveJob(job)
		w.broadcastJobFailed(job)
		return
	}

	job.SetOutputPath(outputDir)
	job.ChapterFiles = chapterFiles
	w.saveJob(job)

	// Build M4B file
	job.SetStatus(JobStatusBuilding)
	w.saveJob(job)
	w.broadcastJobUpdate(job)

	m4bFile, err := w.buildM4B(job, book, chapterFiles)
	if err != nil {
		logger.Warn("M4B build failed: %v (chapter files still available)", err)
		// Don't fail the job, chapter files are still available
	} else {
		job.M4BFile = m4bFile
		logger.Info("M4B file created: %s", m4bFile)

		// Upload to Audiobookshelf if configured
		if w.cfg.AudiobookshelfURL != "" && m4bFile != "" {
			job.SetStatus(JobStatusUploading)
			w.saveJob(job)
			w.broadcastJobUpdate(job)

			if err := w.uploadToAudiobookshelf(job, book); err != nil {
				logger.Warn("Audiobookshelf upload failed: %v", err)
				// Don't fail the job, M4B file is still available locally
			} else {
				logger.Info("Uploaded to Audiobookshelf: %s", book.Title)
			}
		}
	}

	// Mark as completed (if we got here, all chapters succeeded)
	job.SetStatus(JobStatusCompleted)
	job.SetConversionProgress(1.0, "", len(book.Chapters))
	job.SetBuildProgress(1.0)
	w.saveJob(job)
	w.broadcastJobCompleted(job)

	logger.Info("Job %s completed: %s", job.ID, outputDir)
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

// ChapterResult holds the result of converting a single chapter
type ChapterResult struct {
	Index      int
	OutputPath string
	Error      error
}

// convertBook converts the book to audio files using parallel processing
func (w *Worker) convertBook(job *Job, book *parser.Book) (string, []string, error) {
	// Create output directory (use absolute path)
	outputDir := filepath.Join(w.cfg.OutputDir, sanitizeFileName(book.Title))
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get absolute path: %v", err)
	}
	outputDir = absOutputDir
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Set output path early so it can be cleaned up if job is cancelled
	job.SetOutputPath(outputDir)

	totalChapters := len(book.Chapters)

	// Determine number of workers based on provider-specific setting
	numWorkers := w.cfg.GetTTSWorkersForProvider(job.Provider)
	if numWorkers > totalChapters {
		numWorkers = totalChapters
	}

	logger.Info("Converting %d chapters using %d parallel workers (provider: %s)", totalChapters, numWorkers, job.Provider)

	// Results channel and slice
	results := make([]ChapterResult, totalChapters)
	var resultsMu sync.Mutex
	var completedCount int32

	// Fail-fast: cancel all workers when any chunk fails
	failFast := make(chan struct{})
	var failFastOnce sync.Once
	var firstError error
	var firstErrorChapter int

	// Worker ID assignment using a pool
	workerPool := make(chan int, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workerPool <- i
	}

	// Initialize worker progress tracking
	job.InitWorkerProgress(numWorkers)

	// Create job dispatcher
	jd := utils.NewJobDispatcher(numWorkers)

	// Add all chapters as jobs
	for i := range book.Chapters {
		chapterIndex := i
		chapter := book.Chapters[i]

		jd.AddJob(i, func(idx int, ch parser.Chapter) {
			// Get a worker ID from the pool
			workerID := <-workerPool
			defer func() { workerPool <- workerID }()

			// Check for cancellation or fail-fast
			select {
			case <-w.ctx.Done():
				resultsMu.Lock()
				results[idx] = ChapterResult{Index: idx, Error: fmt.Errorf("cancelled")}
				resultsMu.Unlock()
				return
			case <-failFast:
				resultsMu.Lock()
				results[idx] = ChapterResult{Index: idx, Error: fmt.Errorf("cancelled due to earlier failure")}
				resultsMu.Unlock()
				return
			default:
			}

			result := w.convertSingleChapter(job, ch, idx, outputDir, workerID)

			resultsMu.Lock()
			results[idx] = result

			// If this chapter failed, trigger fail-fast to stop all other workers
			if result.Error != nil {
				failFastOnce.Do(func() {
					firstError = result.Error
					firstErrorChapter = idx + 1
					close(failFast)
				})
			}

			completedCount++
			currentCompleted := completedCount
			resultsMu.Unlock()

			// Update progress
			progress := float64(currentCompleted) / float64(totalChapters)
			job.SetConversionProgress(progress, ch.Title, int(currentCompleted))
			w.broadcastJobProgress(job)

			// Save to database periodically (every 5 chapters or 10% progress)
			if currentCompleted%5 == 0 || progress >= 0.1 && int(progress*10) > int((progress-0.1)*10) {
				w.saveJob(job)
			}

			logger.Debug("Converted chapter %d/%d: %s (worker %d)", currentCompleted, totalChapters, ch.Title, workerID)
		}, chapterIndex, chapter)
	}

	// Start parallel processing
	jd.Start()

	// Check for cancellation
	select {
	case <-w.ctx.Done():
		return "", nil, fmt.Errorf("conversion cancelled")
	default:
	}

	// Check if fail-fast was triggered (any chunk failed after retries)
	select {
	case <-failFast:
		chapterTitle := ""
		if firstErrorChapter > 0 && firstErrorChapter <= len(book.Chapters) {
			chapterTitle = book.Chapters[firstErrorChapter-1].Title
		}
		job.AddFailedChapter(firstErrorChapter, chapterTitle, firstError)
		return "", nil, fmt.Errorf("chapter %d (%s) failed: %v", firstErrorChapter, chapterTitle, firstError)
	default:
	}

	// Collect successful results in order
	var chapterFiles []string
	for i := 0; i < totalChapters; i++ {
		if results[i].OutputPath != "" {
			chapterFiles = append(chapterFiles, results[i].OutputPath)
		}
	}

	// Sort by index to maintain chapter order
	sort.Slice(chapterFiles, func(i, j int) bool {
		return chapterFiles[i] < chapterFiles[j]
	})

	return outputDir, chapterFiles, nil
}

// convertSingleChapter converts a single chapter to audio
func (w *Worker) convertSingleChapter(job *Job, chapter parser.Chapter, index int, outputDir string, workerID int) ChapterResult {
	result := ChapterResult{Index: index}

	content := chapter.Content

	// Apply number normalization if provider needs it
	if w.cfg.NeedsNormalization(job.Provider) {
		lang := extractLanguageCode(job.Voice)
		content = w.normalizer.Process(content, lang)
		logger.Debug("Applied number normalization for provider %s (lang: %s)", job.Provider, lang)
	}

	// Apply text sanitization (pronunciation rules) to chapter content
	content = w.sanitizer.Sanitize(content)

	// Save chapter text file for debugging
	textFileName := fmt.Sprintf("%02d_%s.txt", index+1, sanitizeFileName(chapter.Title))
	textFilePath := filepath.Join(outputDir, textFileName)
	if err := os.WriteFile(textFilePath, []byte(content), 0644); err != nil {
		logger.Warn("Failed to save chapter text file '%s': %v", textFileName, err)
	}

	// Progress callback for per-chunk updates
	progressCb := func(chunkIndex, totalChunks int, chunkText string) {
		job.SetWorkerProgress(workerID, index, chapter.Title, chunkIndex+1, totalChunks)
		w.broadcastJobProgress(job)
		w.saveJob(job) // Save to database so TUI can see worker progress
	}

	// Convert chapter with progress tracking
	reader, err := w.ttsService.ConvertToSpeechWithProgress(content, &tts.ConversionOptions{
		Voice:    job.Voice,
		Provider: job.Provider,
		Speed:    job.Speed,
		Pitch:    job.Pitch,
	}, progressCb)
	if err != nil {
		result.Error = fmt.Errorf("TTS conversion failed: %v", err)
		return result
	}

	// Save audio file
	chapterFileName := fmt.Sprintf("%02d_%s.wav", index+1, sanitizeFileName(chapter.Title))
	outputPath := filepath.Join(outputDir, chapterFileName)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create output file: %v", err)
		return result
	}
	defer outputFile.Close()

	if _, err := outputFile.ReadFrom(reader); err != nil {
		result.Error = fmt.Errorf("failed to write audio: %v", err)
		return result
	}

	// Mark worker as done with this chapter
	job.ClearWorkerProgress(workerID)

	result.OutputPath = outputPath
	return result
}

// buildM4B creates M4B audiobook file(s) from chapter audio files
// Returns the primary M4B file path and updates job.M4BFiles with all parts
func (w *Worker) buildM4B(job *Job, book *parser.Book, chapterFiles []string) (string, error) {
	if len(chapterFiles) == 0 {
		return "", fmt.Errorf("no chapter files to build M4B from")
	}

	// Check if ffmpeg is available
	if err := audio.CheckFFmpegAvailable(); err != nil {
		return "", fmt.Errorf("ffmpeg not available: %v", err)
	}

	// Prepare chapter titles
	chapterTitles := make([]string, len(book.Chapters))
	for i, ch := range book.Chapters {
		chapterTitles[i] = ch.Title
	}

	// Split into parts if needed based on estimated M4B output size
	parts, err := audio.SplitIntoPartsByEstimatedSize(chapterFiles, chapterTitles, w.cfg.MaxFileSizeMB, w.cfg.BitRateKbs)
	if err != nil {
		return "", fmt.Errorf("failed to split into parts: %v", err)
	}

	if len(parts) > 1 {
		logger.Info("Book will be split into %d parts", len(parts))
	}

	// Prepare M4B options
	gapDuration := time.Duration(w.cfg.ChapterGapSeconds) * time.Second
	options := audio.M4BOptions{
		Title:           book.Title,
		Author:          book.Author,
		Album:           book.Title,
		Genre:           "Audiobook",
		Description:     book.Description,
		BitRate:         fmt.Sprintf("%dk", w.cfg.BitRateKbs),
		SampleRate:      w.cfg.SampleRateHz,
		GapBetweenChaps: gapDuration,
	}

	// Add cover image if available
	if len(book.CoverImage) > 0 {
		options.CoverImage = book.CoverImage
		options.CoverImageType = book.CoverImageType
	}

	// Determine number of encoder workers
	numEncoders := w.cfg.ConcurrentEncoders
	if numEncoders <= 0 {
		numEncoders = 2 // Default
	}

	if len(parts) > 1 {
		logger.Info("Building %d M4B parts using %d parallel encoders", len(parts), numEncoders)
	}

	// Reset progress for building phase and initialize encoder progress
	job.SetBuildProgress(0)
	job.InitWorkerProgress(numEncoders) // Reuse worker progress for encoders
	w.saveJob(job)                      // Save initial build state to database
	w.broadcastJobProgress(job)

	// Track last saved progress to avoid too many DB writes
	var lastSavedBuildProgress float64

	// Per-encoder progress callback for M4B building
	encoderCb := func(encoderID int, partNum int, totalParts int, progress float64) {
		// Update encoder-specific progress
		job.SetWorkerProgress(encoderID, partNum-1, fmt.Sprintf("Part %d", partNum), int(progress*100), 100)

		// Calculate overall progress across all parts
		// Sum up progress from all active encoders
		job.mu.RLock()
		var totalProgress float64
		activeCount := 0
		for _, wp := range job.WorkerProgress {
			if wp.Active {
				totalProgress += wp.Progress
				activeCount++
			}
		}
		job.mu.RUnlock()

		// Overall progress is based on completed parts + current encoder progress
		overallProgress := totalProgress / float64(len(parts))
		job.SetBuildProgress(overallProgress)
		w.broadcastJobProgress(job)

		// Save to database periodically (every 10% progress change)
		if overallProgress-lastSavedBuildProgress >= 0.1 {
			w.saveJob(job)
			lastSavedBuildProgress = overallProgress
		}
	}

	// Build M4B file(s) in parallel with per-encoder progress tracking
	baseFileName := sanitizeFileName(book.Author + " - " + book.Title)
	m4bFiles, err := audio.BuildMultiPartM4BWithEncoderProgress(parts, job.OutputPath, baseFileName, options, numEncoders, encoderCb)
	if err != nil {
		return "", fmt.Errorf("failed to build M4B: %v", err)
	}

	// Store all M4B files in job
	job.M4BFiles = m4bFiles

	// Return primary file (first part or single file)
	if len(m4bFiles) > 0 {
		return m4bFiles[0], nil
	}
	return "", fmt.Errorf("no M4B files created")
}

// uploadToAudiobookshelf uploads the M4B file(s) to Audiobookshelf server
func (w *Worker) uploadToAudiobookshelf(job *Job, book *parser.Book) error {
	if w.cfg.AudiobookshelfURL == "" {
		return fmt.Errorf("audiobookshelf URL not configured")
	}

	// Use M4BFiles if available (multi-part), otherwise fall back to single M4BFile
	filesToUpload := job.M4BFiles
	if len(filesToUpload) == 0 && job.M4BFile != "" {
		filesToUpload = []string{job.M4BFile}
	}
	if len(filesToUpload) == 0 {
		return fmt.Errorf("no M4B files to upload")
	}

	// Create client and login
	client := audiobookshelf.NewClient(w.cfg.AudiobookshelfURL)
	if err := client.Login(w.cfg.AudiobookshelfUser, w.cfg.AudiobookshelfPassword); err != nil {
		return fmt.Errorf("failed to login to Audiobookshelf: %v", err)
	}

	// Get libraries
	libraries, err := client.GetLibraries()
	if err != nil {
		return fmt.Errorf("failed to get libraries: %v", err)
	}

	// Find target library
	libraryID, err := client.GetLibraryID(libraries, w.cfg.AudiobookshelfLibrary)
	if err != nil {
		return fmt.Errorf("failed to find library '%s': %v", w.cfg.AudiobookshelfLibrary, err)
	}

	// Get folders for the library
	folders, err := client.GetFolders(libraries, w.cfg.AudiobookshelfLibrary)
	if err != nil {
		return fmt.Errorf("failed to get folders: %v", err)
	}

	if len(folders) == 0 {
		return fmt.Errorf("no folders found in library '%s'", w.cfg.AudiobookshelfLibrary)
	}

	// Use first folder
	folderID := folders[0].ID

	// Prepare audiobook for upload
	ab := &audiobookshelf.Audiobook{
		Title:  book.Title,
		Author: book.Author,
		Files:  filesToUpload,
	}

	// Upload with progress callback (no logging to avoid log spam)
	progressCallback := func(fileID int, fileName string, size int64, pos int64, percent int) {
		// Progress is tracked via job status, no need to log each update
	}

	if err := client.UploadBook(ab, libraryID, folderID, progressCallback); err != nil {
		return fmt.Errorf("failed to upload: %v", err)
	}

	// Trigger library scan
	if err := client.ScanLibrary(libraryID); err != nil {
		logger.Warn("Failed to trigger library scan: %v", err)
		// Don't fail, upload was successful
	}

	return nil
}

// sanitizeFileName removes or replaces characters that are invalid in file names
// or problematic for ffmpeg/shell on different operating systems
func sanitizeFileName(name string) string {
	return sanitize.FileName(name)
}

// extractLanguageCode extracts the ISO 639-1 language code from a voice ID.
// Examples: "en-US" -> "en", "ru-RU" -> "ru", "en_US_wavenet" -> "en"
func extractLanguageCode(voice string) string {
	if voice == "" {
		return "en" // Default to English
	}

	// Handle formats like "en-US", "ru-RU", "en_US"
	voice = strings.ToLower(voice)

	// Try splitting by common separators
	for _, sep := range []string{"-", "_"} {
		parts := strings.Split(voice, sep)
		if len(parts) >= 1 && len(parts[0]) == 2 {
			return parts[0]
		}
	}

	// If voice is already a 2-letter code
	if len(voice) == 2 {
		return voice
	}

	// Default to English
	return "en"
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
