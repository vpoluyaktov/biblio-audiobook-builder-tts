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

	"biblio-audiobook-builder-tts/internal/audio"
	"biblio-audiobook-builder-tts/internal/audiobookshelf"
	"biblio-audiobook-builder-tts/internal/config"
	"biblio-audiobook-builder-tts/internal/logger"
	"biblio-audiobook-builder-tts/internal/normalize"
	"biblio-audiobook-builder-tts/internal/parser"
	"biblio-audiobook-builder-tts/internal/sanitize"
	"biblio-audiobook-builder-tts/internal/ssml"
	"biblio-audiobook-builder-tts/internal/storage"
	"biblio-audiobook-builder-tts/internal/stress"
	"biblio-audiobook-builder-tts/internal/tts"
	"biblio-audiobook-builder-tts/internal/utils"
)

// JobDB defines the database operations needed for job management
type JobDB interface {
	GetJob(id string) (*storage.Job, error)
	GetPendingJob() (*storage.Job, error)
	CreateJob(job *storage.Job) error
	UpdateJob(job *storage.Job) error
	DeleteJob(id string) error
	GetAllPronunciationRules() ([]storage.PronunciationRule, error)
}

// Worker processes conversion jobs from the queue
type Worker struct {
	db                JobDB
	hub               *Hub
	ttsService        tts.Service
	cfg               *config.Config
	sanitizer         *sanitize.TextSanitizer
	normalizer        *normalize.Processor
	separatorDetector *normalize.PartSeparatorDetector
	stressClient      *stress.Client
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// NewWorker creates a new job worker
func NewWorker(db JobDB, hub *Hub, ttsService tts.Service, cfg *config.Config) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize text sanitizer with pronunciation dictionary
	textSanitizer := sanitize.NewTextSanitizer()

	// Load default rules if enabled
	logger.Info("UseDefaultPronunciation setting: %v", cfg.UseDefaultPronunciation)
	if cfg.UseDefaultPronunciation {
		textSanitizer.LoadDefaultRules()
		ruleCount := textSanitizer.GetDictionary().RuleCount()
		logger.Info("Loaded %d default pronunciation rules from built-in CSV files", ruleCount)
	} else {
		logger.Warn("Default pronunciation rules NOT loaded (UseDefaultPronunciation is false)")
	}

	// Note: Database pronunciation rules are loaded per-chapter to allow
	// real-time updates without service restart

	// Initialize part separator detector if enabled
	var separatorDetector *normalize.PartSeparatorDetector
	if cfg.DetectPartSeparators {
		separatorDetector = normalize.NewPartSeparatorDetector()
		logger.Info("Part separator detection enabled")
	}

	// Initialize stress client if URL is configured
	var stressClient *stress.Client
	if cfg.StressServerURL != "" {
		stressClient = stress.NewClient(cfg.StressServerURL)
		if err := stressClient.CheckHealth(); err != nil {
			logger.Warn("Stress server not available at %s: %v", cfg.StressServerURL, err)
		} else {
			logger.Info("Stress server connected at %s", cfg.StressServerURL)
		}
	}

	return &Worker{
		db:                db,
		hub:               hub,
		ttsService:        ttsService,
		cfg:               cfg,
		sanitizer:         textSanitizer,
		normalizer:        normalize.NewProcessor(),
		separatorDetector: separatorDetector,
		stressClient:      stressClient,
		ctx:               ctx,
		cancel:            cancel,
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

// loadPronunciationRulesFromDB reloads pronunciation rules from database
func (w *Worker) loadPronunciationRulesFromDB() error {
	// Get all pronunciation rules from database
	dbRules, err := w.db.GetAllPronunciationRules()
	if err != nil {
		return fmt.Errorf("failed to get pronunciation rules: %w", err)
	}

	// Clear existing database rules (keep default and file-based rules)
	// We need to reload only the database rules, so we'll clear all and reload everything
	dict := w.sanitizer.GetDictionary()
	dict.Clear()

	// Reload default rules if enabled
	if w.cfg.UseDefaultPronunciation {
		w.sanitizer.LoadDefaultRules()
	}

	// Load database rules
	for _, rule := range dbRules {
		if err := dict.AddRuleWithSSML(
			rule.Pattern,
			rule.ReplacementPlain,
			rule.ReplacementSSML,
			rule.Language,
			rule.Enabled,
		); err != nil {
			logger.Warn("Failed to add pronunciation rule from database: %v", err)
		}
	}

	totalRules := dict.RuleCount()
	logger.Debug("Reloaded pronunciation rules: %d from database, %d total rules", len(dbRules), totalRules)
	return nil
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

	// Reload pronunciation rules from database once per job (before parallel
	// chapter processing). Loading per-chapter would race: parallel workers
	// share one PronunciationDictionary, and Clear()+reload in one goroutine
	// would wipe rules out from under another goroutine's Apply call.
	if err := w.loadPronunciationRulesFromDB(); err != nil {
		logger.Warn("Failed to reload pronunciation rules from database: %v", err)
	}

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
	return parser.ParseFile(filePath)
}

// ChapterResult holds the result of converting a single chapter
type ChapterResult struct {
	Index      int
	OutputPath string
	Error      error
	Skipped    bool // True if chapter was skipped (no speakable content for target language)
}

// convertBook converts the book to audio files using parallel processing
func (w *Worker) convertBook(job *Job, book *parser.Book) (string, []string, error) {
	// Create output directory inside temp dir (use absolute path)
	outputDir := filepath.Join(w.cfg.TempDir, sanitizeFileName(book.Title))
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
	numWorkers := 3 // Default
	if providerInfo := w.ttsService.GetProviderInfo(job.Provider); providerInfo != nil && providerInfo.TTSWorkers > 0 {
		numWorkers = providerInfo.TTSWorkers
	}
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

			if result.Skipped {
				logger.Debug("Skipped chapter %d/%d: %s (no speakable content, worker %d)", currentCompleted, totalChapters, ch.Title, workerID)
			} else {
				logger.Debug("Converted chapter %d/%d: %s (worker %d)", currentCompleted, totalChapters, ch.Title, workerID)
			}
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

	// Get language for language-specific processing
	lang := job.Language
	if lang == "" {
		lang = "en" // Fallback to English if not set
	}

	// Reload pronunciation rules from database to get latest changes
	if err := w.loadPronunciationRulesFromDB(); err != nil {
		logger.Warn("Failed to reload pronunciation rules from database: %v", err)
	}

	// Get provider info BEFORE applying pronunciation rules to determine SSML support
	needsNormalization := true // Default to true for safety
	transliterationEnabled := false
	stressEnabled := false
	useSSML := false
	if providerInfo := w.ttsService.GetProviderInfo(job.Provider); providerInfo != nil {
		needsNormalization = providerInfo.NormalizeNumbers
		transliterationEnabled = providerInfo.Transliteration
		stressEnabled = providerInfo.StressEnabled
		useSSML = providerInfo.SSMLSupport
	}

	// Step 1: Apply text sanitization and pronunciation dictionary rules FIRST
	// This ensures user-defined pronunciation rules have priority over automatic processing
	// Apply only rules for the book's language, using SSML replacements if provider supports it
	content = w.sanitizer.SanitizeWithOptions(content, lang, useSSML)

	// Step 2: Apply number normalization if provider needs it
	// IMPORTANT: This must run BEFORE Latin-to-Russian transliteration because
	// Roman numerals (I, II, III, IV, etc.) are Latin characters that the
	// transliterator would convert letter-by-letter (e.g., "III" → "ай ай ай")
	// before the normalizer could process them (e.g., "Глава III" → "Глава третья").
	if needsNormalization {
		lang := job.Language
		if lang == "" {
			lang = "en" // Fallback to English if not set
		}
		content = w.normalizer.Process(content, lang)
		logger.Debug("Applied number normalization for provider %s (lang: %s)", job.Provider, lang)
	}

	// Step 3: Apply Latin-to-Russian transliteration if enabled (for Russian text only)
	// Runs after normalization so Roman numerals are already converted to Cyrillic words.
	if transliterationEnabled && lang == "ru" {
		content = w.sanitizer.ConvertLatinToRussian(content)
		logger.Debug("Applied Latin-to-Russian transliteration for provider %s", job.Provider)
	}

	// Step 4: Apply stress marking for Russian text if enabled for this provider
	if stressEnabled && w.stressClient != nil && w.stressClient.IsAvailable() {
		lang := job.Language
		if lang == "" {
			lang = "en"
		}
		// Only apply stress marking for Russian language
		// Process sentence by sentence to ensure correct homograph disambiguation
		if lang == "ru" && w.stressClient.SupportsLanguage(lang) {
			stressedContent, err := w.stressClient.AddStressToSentences(content, lang)
			if err != nil {
				logger.Warn("Failed to add stress markers: %v (continuing without stress)", err)
			} else {
				content = stressedContent
				logger.Debug("Applied stress marking for provider %s (lang: %s)", job.Provider, lang)
			}
		}
	}

	// Detect and mark part separators if enabled
	if w.separatorDetector != nil {
		content = w.separatorDetector.DetectAndMark(content)
	}

	// Note: SSML wrapping is applied per-chunk in the TTS adapter, not here
	// This allows proper chunking of long chapters before SSML tags are added

	// Save chapter text file for debugging
	textFileName := fmt.Sprintf("%04d_%s.txt", index+1, sanitizeFileName(chapter.Title))
	textFilePath := filepath.Join(outputDir, textFileName)
	if err := os.WriteFile(textFilePath, []byte(content), 0644); err != nil {
		logger.Warn("Failed to save chapter text file '%s': %v", textFileName, err)
	}

	// Check if chapter has speakable content for the target language
	// Generate silent audio for chapters that have no content the TTS engine can process
	// (e.g., "Illustration." for Russian TTS) to maintain chapter alignment in the audiobook
	if !sanitize.HasSpeakableContentForLanguage(content, lang) {
		logger.Info("Chapter %d (%s) has no speakable content for language '%s' - generating 1 second of silence", index+1, chapter.Title, lang)

		// Generate a 1-second silent WAV file to maintain chapter alignment
		chapterFileName := fmt.Sprintf("%04d_%s.wav", index+1, sanitizeFileName(chapter.Title))
		outputPath := filepath.Join(outputDir, chapterFileName)

		// Get provider's sample rate, default to 48000 if not available
		sampleRate := 48000
		if providerInfo := w.ttsService.GetProviderInfo(job.Provider); providerInfo != nil {
			// Cast db to *storage.DB to access GetProvider method
			if db, ok := w.db.(*storage.DB); ok {
				if dbProvider, err := db.GetProvider(providerInfo.ID); err == nil && dbProvider != nil && dbProvider.SampleRate > 0 {
					sampleRate = dbProvider.SampleRate
				}
			}
		}

		if err := audio.GenerateSilentWAV(1*time.Second, outputPath, sampleRate); err != nil {
			result.Error = fmt.Errorf("failed to generate silent audio: %v", err)
			return result
		}

		result.OutputPath = outputPath
		result.Skipped = true // Still mark as skipped for logging purposes
		return result
	}

	// Check if provider supports SSML
	ssmlSupport := false
	if providerInfo := w.ttsService.GetProviderInfo(job.Provider); providerInfo != nil {
		ssmlSupport = providerInfo.SSMLSupport
	}

	// Save SSML debug file if SSML is enabled (before chunking)
	if ssmlSupport {
		ssmlText := ssml.AddSSMLBreaks(content, ssml.SSMLOptions{
			SentenceBreakMs:       w.cfg.SentenceBreakMs,
			ParagraphBreakMs:      w.cfg.ParagraphBreakMs,
			ConvertDashesToBreaks: w.cfg.ConvertDashesToBreaks,
			DashBreakDurationMs:   w.cfg.DashBreakDurationMs,
			ParenthesesBreakMs:    w.cfg.ParenthesesBreakMs,
			ColonBreakMs:          w.cfg.ColonBreakMs,
			TitleBreakMs:          w.cfg.TitleBreakMs,
		})
		ssmlFileName := fmt.Sprintf("%04d_%s.ssml.txt", index+1, sanitizeFileName(chapter.Title))
		ssmlFilePath := filepath.Join(outputDir, ssmlFileName)
		if err := os.WriteFile(ssmlFilePath, []byte(ssmlText), 0644); err != nil {
			logger.Warn("Failed to save SSML debug file '%s': %v", ssmlFileName, err)
		}
	}

	// Check if content has part separators that need silence insertion
	if normalize.HasPartSeparators(content) {
		return w.convertChapterWithParts(job, chapter, content, index, outputDir, workerID, ssmlSupport)
	}

	// Progress callback for per-chunk updates
	progressCb := func(chunkIndex, totalChunks int, chunkText string) {
		job.SetWorkerProgress(workerID, index, chapter.Title, chunkIndex+1, totalChunks)
		w.broadcastJobProgress(job)
		w.saveJob(job) // Save to database so TUI can see worker progress
	}

	// Convert chapter with progress tracking
	reader, err := w.ttsService.ConvertToSpeechWithProgress(content, &tts.ConversionOptions{
		Voice:                 job.Voice,
		Provider:              job.Provider,
		Speed:                 job.Speed,
		Pitch:                 job.Pitch,
		Language:              job.Language,
		SSMLSupport:           ssmlSupport,
		SentenceBreakMs:       w.cfg.SentenceBreakMs,
		ParagraphBreakMs:      w.cfg.ParagraphBreakMs,
		ConvertDashesToBreaks: w.cfg.ConvertDashesToBreaks,
		DashBreakDurationMs:   w.cfg.DashBreakDurationMs,
		ParenthesesBreakMs:    w.cfg.ParenthesesBreakMs,
		ColonBreakMs:          w.cfg.ColonBreakMs,
		TitleBreakMs:          w.cfg.TitleBreakMs,
	}, progressCb)
	if err != nil {
		result.Error = fmt.Errorf("TTS conversion failed: %v", err)
		return result
	}

	// Save audio file
	chapterFileName := fmt.Sprintf("%04d_%s.wav", index+1, sanitizeFileName(chapter.Title))
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

// convertChapterWithParts handles chapters that contain part separators
// It splits the content, generates TTS for each part, inserts silence between parts,
// and concatenates the audio files into a single chapter audio file
func (w *Worker) convertChapterWithParts(job *Job, chapter parser.Chapter, content string, index int, outputDir string, workerID int, ssmlSupport bool) ChapterResult {
	result := ChapterResult{Index: index}

	// Split content at separator markers
	parts := normalize.SplitByMarker(content)
	numParts := len(parts)

	logger.Info("Chapter %d (%s) has %d parts separated by scene breaks", index+1, chapter.Title, numParts)

	// Save SSML debug file if SSML is enabled (before chunking)
	if ssmlSupport {
		ssmlText := ssml.AddSSMLBreaks(content, ssml.SSMLOptions{
			SentenceBreakMs:       w.cfg.SentenceBreakMs,
			ParagraphBreakMs:      w.cfg.ParagraphBreakMs,
			ConvertDashesToBreaks: w.cfg.ConvertDashesToBreaks,
			DashBreakDurationMs:   w.cfg.DashBreakDurationMs,
			TitleBreakMs:          w.cfg.TitleBreakMs,
		})
		ssmlFileName := fmt.Sprintf("%04d_%s.ssml.txt", index+1, sanitizeFileName(chapter.Title))
		ssmlFilePath := filepath.Join(outputDir, ssmlFileName)
		if err := os.WriteFile(ssmlFilePath, []byte(ssmlText), 0644); err != nil {
			logger.Warn("Failed to save SSML debug file '%s': %v", ssmlFileName, err)
		}
	}

	// Get provider's sample rate for silence generation
	sampleRate := 48000
	if providerInfo := w.ttsService.GetProviderInfo(job.Provider); providerInfo != nil {
		if db, ok := w.db.(*storage.DB); ok {
			if dbProvider, err := db.GetProvider(providerInfo.ID); err == nil && dbProvider != nil && dbProvider.SampleRate > 0 {
				sampleRate = dbProvider.SampleRate
			}
		}
	}

	// Determine silence duration between parts
	partGapSeconds := w.cfg.PartGapSeconds
	if partGapSeconds <= 0 {
		partGapSeconds = w.cfg.ChapterGapSeconds // Fall back to chapter gap
	}
	if partGapSeconds <= 0 {
		partGapSeconds = 2.0 // Default 2 seconds
	}

	// Create temp directory for part files
	partsDir := filepath.Join(outputDir, fmt.Sprintf("chapter_%04d_parts", index+1))
	if err := os.MkdirAll(partsDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create parts directory: %v", err)
		return result
	}
	defer os.RemoveAll(partsDir) // Clean up temp parts directory

	// Generate silence file for between parts
	silenceFile := filepath.Join(partsDir, "silence.wav")
	partGapDuration := time.Duration(partGapSeconds * float64(time.Second))
	if err := audio.GenerateSilentWAV(partGapDuration, silenceFile, sampleRate); err != nil {
		result.Error = fmt.Errorf("failed to generate silence file: %v", err)
		return result
	}

	// Process each part and collect audio files
	var audioFiles []string
	lang := job.Language
	if lang == "" {
		lang = "en"
	}

	for partIdx, partContent := range parts {
		// Skip empty parts
		if strings.TrimSpace(partContent) == "" {
			continue
		}

		// Check if part has speakable content
		if !sanitize.HasSpeakableContentForLanguage(partContent, lang) {
			logger.Debug("Part %d of chapter %d has no speakable content, skipping", partIdx+1, index+1)
			continue
		}

		// Progress callback for this part
		progressCb := func(chunkIndex, totalChunks int, chunkText string) {
			// Show progress as "part X of Y, chunk Z of N"
			overallProgress := partIdx*100/numParts + (chunkIndex+1)*100/(numParts*totalChunks)
			job.SetWorkerProgress(workerID, index, fmt.Sprintf("%s (part %d/%d)", chapter.Title, partIdx+1, numParts), overallProgress, 100)
			w.broadcastJobProgress(job)
		}

		// Convert part to speech
		reader, err := w.ttsService.ConvertToSpeechWithProgress(partContent, &tts.ConversionOptions{
			Voice:                 job.Voice,
			Provider:              job.Provider,
			Speed:                 job.Speed,
			Pitch:                 job.Pitch,
			Language:              job.Language,
			SSMLSupport:           ssmlSupport,
			SentenceBreakMs:       w.cfg.SentenceBreakMs,
			ParagraphBreakMs:      w.cfg.ParagraphBreakMs,
			ConvertDashesToBreaks: w.cfg.ConvertDashesToBreaks,
			DashBreakDurationMs:   w.cfg.DashBreakDurationMs,
			TitleBreakMs:          w.cfg.TitleBreakMs,
		}, progressCb)
		if err != nil {
			result.Error = fmt.Errorf("TTS conversion failed for part %d: %v", partIdx+1, err)
			return result
		}

		// Save part audio file
		partFileName := fmt.Sprintf("part_%02d.wav", partIdx+1)
		partPath := filepath.Join(partsDir, partFileName)

		partFile, err := os.Create(partPath)
		if err != nil {
			result.Error = fmt.Errorf("failed to create part file: %v", err)
			return result
		}

		if _, err := partFile.ReadFrom(reader); err != nil {
			partFile.Close()
			result.Error = fmt.Errorf("failed to write part audio: %v", err)
			return result
		}
		partFile.Close()

		// Add part audio to list
		audioFiles = append(audioFiles, partPath)

		// Add silence between parts (not after the last part)
		if partIdx < numParts-1 {
			audioFiles = append(audioFiles, silenceFile)
		}
	}

	// If no audio files were generated, create a silent placeholder
	if len(audioFiles) == 0 {
		chapterFileName := fmt.Sprintf("%04d_%s.wav", index+1, sanitizeFileName(chapter.Title))
		outputPath := filepath.Join(outputDir, chapterFileName)
		if err := audio.GenerateSilentWAV(1*time.Second, outputPath, sampleRate); err != nil {
			result.Error = fmt.Errorf("failed to generate silent audio: %v", err)
			return result
		}
		result.OutputPath = outputPath
		result.Skipped = true
		return result
	}

	// Concatenate all parts into final chapter audio
	chapterFileName := fmt.Sprintf("%04d_%s.wav", index+1, sanitizeFileName(chapter.Title))
	outputPath := filepath.Join(outputDir, chapterFileName)

	if err := audio.ConcatWAVFiles(audioFiles, outputPath); err != nil {
		result.Error = fmt.Errorf("failed to concatenate parts: %v", err)
		return result
	}

	// Mark worker as done with this chapter
	job.ClearWorkerProgress(workerID)

	logger.Info("Chapter %d (%s) converted with %d parts and silence gaps", index+1, chapter.Title, numParts)

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
	// Use default 128kbps for size estimation (actual bitrate is determined by ffmpeg)
	parts, err := audio.SplitIntoPartsByEstimatedSize(chapterFiles, chapterTitles, w.cfg.MaxFileSizeMB, 128)
	if err != nil {
		return "", fmt.Errorf("failed to split into parts: %v", err)
	}

	if len(parts) > 1 {
		logger.Info("Book will be split into %d parts", len(parts))
	}

	// Prepare M4B options (sample rate is auto-detected from source files)
	gapDuration := time.Duration(w.cfg.ChapterGapSeconds * float64(time.Second))
	logger.Info("Building M4B with chapter gap: %v (ChapterGapSeconds=%.2f)", gapDuration, w.cfg.ChapterGapSeconds)

	// Resolve genre with three-tier fallback:
	// 1. job.BookGenre (from OPDS categories or previous SetBook call)
	// 2. book.Genre (from ebook file metadata)
	// 3. "Audiobook" (hardcoded default)
	genre := job.BookGenre
	if genre == "" && book.Genre != "" {
		genre = book.Genre
	}
	if genre == "" {
		genre = "Audiobook"
	}

	options := audio.M4BOptions{
		Title:           book.Title,
		Author:          book.Author,
		Album:           book.Title,
		Series:          book.Series,
		SeriesNumber:    book.SeriesNumber,
		Genre:           genre,
		Description:     book.Description,
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
		Series: book.Series,
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
