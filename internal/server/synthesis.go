package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"biblio-audiobook-builder-tts/internal/audio"
	"biblio-audiobook-builder-tts/internal/logger"
	"biblio-audiobook-builder-tts/internal/normalize"
	"biblio-audiobook-builder-tts/internal/parser"
	"biblio-audiobook-builder-tts/internal/sanitize"
	"biblio-audiobook-builder-tts/internal/ssml"
	"biblio-audiobook-builder-tts/internal/storage"
	"biblio-audiobook-builder-tts/internal/tts"
)

// Text processing and speech synthesis shared by book conversion and voice tests

// textProcessing holds the provider-specific text processing switches
type textProcessing struct {
	normalizeNumbers bool
	transliterate    bool
	stress           bool
	ssml             bool
}

func (w *Worker) textProcessing(provider string) textProcessing {
	info := w.ttsService.GetProviderInfo(provider)
	if info == nil {
		return textProcessing{normalizeNumbers: true} // Default to true for safety
	}
	return textProcessing{
		normalizeNumbers: info.NormalizeNumbers,
		transliterate:    info.Transliteration,
		stress:           info.StressEnabled,
		ssml:             info.SSMLSupport,
	}
}

// prepareText applies all text processing that precedes speech synthesis
func (w *Worker) prepareText(sanitizer *sanitize.TextSanitizer, content, lang, provider string, processing textProcessing) string {
	// Step 1: Apply text sanitization and pronunciation dictionary rules FIRST
	// This ensures user-defined pronunciation rules have priority over automatic processing
	// Apply only rules for the book's language, using SSML replacements if provider supports it
	content = sanitizer.SanitizeWithOptions(content, lang, processing.ssml)

	// Step 2: Apply number normalization if provider needs it
	// IMPORTANT: This must run BEFORE Latin-to-Russian transliteration because
	// Roman numerals (I, II, III, IV, etc.) are Latin characters that the
	// transliterator would convert letter-by-letter (e.g., "III" → "ай ай ай")
	// before the normalizer could process them (e.g., "Глава III" → "Глава третья").
	if processing.normalizeNumbers {
		content = w.normalizer.Process(content, lang)
		logger.Debug("Applied number normalization for provider %s (lang: %s)", provider, lang)
	}

	// Step 3: Apply Latin-to-Russian transliteration if enabled (for Russian text only)
	// Runs after normalization so Roman numerals are already converted to Cyrillic words.
	if processing.transliterate && lang == "ru" {
		content = sanitizer.ConvertLatinToRussian(content)
		logger.Debug("Applied Latin-to-Russian transliteration for provider %s", provider)
	}

	// Step 4: Apply stress marking for Russian text if enabled for this provider
	// Process sentence by sentence to ensure correct homograph disambiguation
	if processing.stress && lang == "ru" && w.stressClient != nil &&
		w.stressClient.IsAvailable() && w.stressClient.SupportsLanguage(lang) {
		stressedContent, err := w.stressClient.AddStressToSentences(content, lang)
		if err != nil {
			logger.Warn("Failed to add stress markers: %v (continuing without stress)", err)
		} else {
			content = stressedContent
			logger.Debug("Applied stress marking for provider %s (lang: %s)", provider, lang)
		}
	}

	// Detect and mark part separators if enabled
	if w.separatorDetector != nil {
		content = w.separatorDetector.DetectAndMark(content)
	}

	return content
}

// conversionOptions builds TTS options from the voice parameters and the saved pause settings
func (w *Worker) conversionOptions(provider, voice, language string, speed, pitch float64, ssmlSupport bool) *tts.ConversionOptions {
	return &tts.ConversionOptions{
		Voice:            voice,
		Provider:         provider,
		Speed:            speed,
		Pitch:            pitch,
		Language:         language,
		SSMLSupport:      ssmlSupport,
		SentenceBreakMs:  w.cfg.SentenceBreakMs,
		ParagraphBreakMs: w.cfg.ParagraphBreakMs,
		// A zero duration disables dash breaks; the SSML builder would otherwise apply its default
		ConvertDashesToBreaks: w.cfg.ConvertDashesToBreaks && w.cfg.DashBreakDurationMs > 0,
		DashBreakDurationMs:   w.cfg.DashBreakDurationMs,
		ParenthesesBreakMs:    w.cfg.ParenthesesBreakMs,
		ColonBreakMs:          w.cfg.ColonBreakMs,
		CommaBreakMs:          w.cfg.CommaBreakMs,
		TitleBreakMs:          w.cfg.TitleBreakMs,
	}
}

// saveSSMLDebugFile saves the chapter text with SSML breaks (before chunking)
func (w *Worker) saveSSMLDebugFile(content string, opts *tts.ConversionOptions, outputDir string, index int, title string) {
	if !opts.SSMLSupport {
		return
	}
	ssmlFileName := fmt.Sprintf("%04d_%s.ssml.txt", index+1, sanitizeFileName(title))
	ssmlText := ssml.AddSSMLBreaks(content, opts.SSMLOptions())
	if err := os.WriteFile(filepath.Join(outputDir, ssmlFileName), []byte(ssmlText), 0644); err != nil {
		logger.Warn("Failed to save SSML debug file '%s': %v", ssmlFileName, err)
	}
}

func (w *Worker) providerSampleRate(provider string) int {
	if info := w.ttsService.GetProviderInfo(provider); info != nil {
		if db, ok := w.db.(*storage.DB); ok {
			if dbProvider, err := db.GetProvider(info.ID); err == nil && dbProvider != nil && dbProvider.SampleRate > 0 {
				return dbProvider.SampleRate
			}
		}
	}
	return 48000
}

func (w *Worker) partGap() time.Duration {
	seconds := w.cfg.PartGapSeconds
	if seconds <= 0 {
		seconds = w.cfg.ChapterGapSeconds // Fall back to chapter gap
	}
	if seconds <= 0 {
		seconds = 2.0
	}
	return time.Duration(seconds * float64(time.Second))
}

// synthesizeParts converts text containing part separators into outputPath, inserting
// silence between the parts. It writes one second of silence and reports false if no
// part has speakable content.
func (w *Worker) synthesizeParts(content string, opts *tts.ConversionOptions, partsDir, outputPath string,
	progressCb func(partIdx, numParts, chunkIndex, totalChunks int)) (bool, error) {
	parts := normalize.SplitByMarker(content)
	numParts := len(parts)
	sampleRate := w.providerSampleRate(opts.Provider)

	if err := os.MkdirAll(partsDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create parts directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(partsDir); err != nil {
			logger.Warn("Failed to remove parts directory '%s': %v", partsDir, err)
		}
	}()

	silenceFile := filepath.Join(partsDir, "silence.wav")
	if err := audio.GenerateSilentWAV(w.partGap(), silenceFile, sampleRate); err != nil {
		return false, fmt.Errorf("failed to generate silence file: %w", err)
	}

	lang := opts.Language
	if lang == "" {
		lang = "en"
	}

	var audioFiles []string
	for partIdx, partContent := range parts {
		if strings.TrimSpace(partContent) == "" {
			continue
		}

		if !sanitize.HasSpeakableContentForLanguage(partContent, lang) {
			logger.Debug("Part %d/%d has no speakable content, skipping", partIdx+1, numParts)
			continue
		}

		var chunkCb tts.ProgressCallback
		if progressCb != nil {
			chunkCb = func(chunkIndex, totalChunks int, _ string) {
				progressCb(partIdx, numParts, chunkIndex, totalChunks)
			}
		}

		reader, err := w.ttsService.ConvertToSpeechWithProgress(partContent, opts, chunkCb)
		if err != nil {
			return false, fmt.Errorf("TTS conversion failed for part %d: %w", partIdx+1, err)
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			return false, fmt.Errorf("failed to read part audio: %w", err)
		}
		partPath := filepath.Join(partsDir, fmt.Sprintf("part_%02d.wav", partIdx+1))
		if err := os.WriteFile(partPath, data, 0644); err != nil {
			return false, fmt.Errorf("failed to write part audio: %w", err)
		}
		audioFiles = append(audioFiles, partPath)

		// Add silence between parts (not after the last part)
		if partIdx < numParts-1 {
			audioFiles = append(audioFiles, silenceFile)
		}
	}

	if len(audioFiles) == 0 {
		if err := audio.GenerateSilentWAV(time.Second, outputPath, sampleRate); err != nil {
			return false, fmt.Errorf("failed to generate silent audio: %w", err)
		}
		return false, nil
	}

	if err := audio.ConcatWAVFiles(audioFiles, outputPath); err != nil {
		return false, fmt.Errorf("failed to concatenate parts: %w", err)
	}

	return true, nil
}

// PreviewSpeech synthesizes test text with the same pause handling as a book chapter.
// Transliteration and stress marking are not applied.
func (w *Worker) PreviewSpeech(req TestVoiceRequest) ([]byte, error) {
	content, err := parser.RenderPlainText(req.Text)
	if err != nil {
		return nil, err
	}

	// A private sanitizer avoids modifying the one used by a running conversion
	dbRules, err := w.db.GetAllPronunciationRules()
	if err != nil {
		return nil, fmt.Errorf("failed to get pronunciation rules: %w", err)
	}
	sanitizer := sanitize.NewTextSanitizer()
	w.applyPronunciationRules(sanitizer, dbRules)

	processing := w.textProcessing(req.Provider)
	processing.transliterate = false
	processing.stress = false
	content = w.prepareText(sanitizer, content, req.Language, req.Provider, processing)

	if !sanitize.HasSpeakableContentForLanguage(content, req.Language) {
		return nil, fmt.Errorf("no speakable content for language '%s'", req.Language)
	}

	opts := w.conversionOptions(req.Provider, req.Voice, req.Language, req.Speed, req.Pitch, processing.ssml)

	if !normalize.HasPartSeparators(content) {
		reader, err := w.ttsService.ConvertToSpeechWithProgress(content, opts, nil)
		if err != nil {
			return nil, err
		}
		return io.ReadAll(reader)
	}

	tempDir, err := os.MkdirTemp("", "voice_test_")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			logger.Warn("Failed to remove temp directory '%s': %v", tempDir, err)
		}
	}()

	outputPath := filepath.Join(tempDir, "test.wav")
	hasAudio, err := w.synthesizeParts(content, opts, filepath.Join(tempDir, "parts"), outputPath, nil)
	if err != nil {
		return nil, err
	}
	if !hasAudio {
		return nil, fmt.Errorf("no speakable content for language '%s'", req.Language)
	}
	return os.ReadFile(outputPath)
}
