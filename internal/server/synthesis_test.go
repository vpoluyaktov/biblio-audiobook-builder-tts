package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"biblio-audiobook-builder-tts/internal/audio"
	"biblio-audiobook-builder-tts/internal/config"
	"biblio-audiobook-builder-tts/internal/parser"
	"biblio-audiobook-builder-tts/internal/storage"
	"biblio-audiobook-builder-tts/internal/tts"
)

// fakeSilero records the SSML sent to the TTS backend
type fakeSilero struct {
	mu    sync.Mutex
	texts []string
}

func (f *fakeSilero) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	texts := f.texts
	f.texts = nil
	return texts
}

func newSynthesisServer(t *testing.T) (*Server, *storage.DB, *fakeSilero) {
	t.Helper()

	wavPath := filepath.Join(t.TempDir(), "chunk.wav")
	if err := audio.GenerateSilentWAV(100*time.Millisecond, wavPath, 48000); err != nil {
		t.Fatal(err)
	}
	wav, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}

	fake := &fakeSilero{}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tts" {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		fake.mu.Lock()
		fake.texts = append(fake.texts, r.URL.Query().Get("text"))
		fake.mu.Unlock()
		_, _ = w.Write(wav)
	}))
	t.Cleanup(backend.Close)

	db, err := storage.NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.InitializeDefaults(); err != nil {
		t.Fatal(err)
	}
	provider, err := db.GetProvider("silero")
	if err != nil || provider == nil {
		t.Fatalf("silero provider: %v", err)
	}
	provider.Enabled = true
	provider.URL = backend.URL
	provider.SSMLSupport = true
	if err := db.UpdateProvider(provider); err != nil {
		t.Fatal(err)
	}

	cfg := loadSavedConfig(t, db)
	cfg.TempDir = t.TempDir()
	s := New(":0", cfg, tts.NewServiceWithDB(cfg, db))
	s.SetDB(db)
	return s, db, fake
}

func loadSavedConfig(t *testing.T, db *storage.DB) *config.Config {
	t.Helper()
	saved, err := db.GetAllConfig()
	if err != nil {
		t.Fatal(err)
	}
	return config.LoadFromDB(saved.ToAppConfig())
}

func postJSON(t *testing.T, handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	return rec
}

var distinctPauses = map[string]any{
	"sentence_break_ms":      110,
	"paragraph_break_ms":     220,
	"dash_break_duration_ms": 330,
	"parentheses_break_ms":   440,
	"colon_break_ms":         550,
	"title_break_ms":         660,
	"chapter_gap_seconds":    1.5,
	"part_gap_seconds":       2.5,
}

const pauseText = "First sentence. Second, sentence\nWait - there (an aside) is this: done"

func testVoice(t *testing.T, s *Server, text string) *httptest.ResponseRecorder {
	t.Helper()
	return postJSON(t, s.handleTestVoice, TestVoiceRequest{
		Provider: "silero", Voice: "silero:v3_en#en_0", Language: "en", Text: text,
	})
}

func convertAsBook(t *testing.T, s *Server, text string) ChapterResult {
	t.Helper()
	content, err := parser.RenderPlainText(text)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.worker.loadPronunciationRulesFromDB(); err != nil {
		t.Fatal(err)
	}
	job := NewJob("book.epub", "", "silero", "silero:v3_en#en_0", "en", "", 1, 1, true)
	result := s.worker.convertSingleChapter(job, parser.Chapter{Title: "One", Content: content}, 0, t.TempDir(), 0)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	return result
}

func TestVoiceTestMatchesConversion(t *testing.T) {
	s, _, fake := newSynthesisServer(t)
	postJSON(t, s.saveSettings, distinctPauses)

	rec := testVoice(t, s, pauseText)
	preview := fake.sent()
	convertAsBook(t, s, pauseText)
	book := fake.sent()

	got := strings.Join(preview, "|")
	if want := strings.Join(book, "|"); got != want {
		t.Errorf("voice test SSML differs from conversion:\n test: %s\n book: %s", got, want)
	}
	for _, ms := range []string{"110", "220", "330", "440", "550"} {
		if !strings.Contains(got, `<break time="`+ms+`ms"/>`) {
			t.Errorf("missing %sms break in %s", ms, got)
		}
	}
	if ct := rec.Header().Get("Content-Type"); ct != "audio/wav" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestVoiceTestSceneSeparator(t *testing.T) {
	s, _, fake := newSynthesisServer(t)
	postJSON(t, s.saveSettings, distinctPauses)
	const text = "Before (an aside) this: now.\n---\nAfter the break."

	rec := testVoice(t, s, text)
	preview := fake.sent()
	result := convertAsBook(t, s, text)
	book := fake.sent()

	if len(preview) != 2 || strings.Join(preview, "|") != strings.Join(book, "|") {
		t.Fatalf("unexpected requests:\n test: %q\n book: %q", preview, book)
	}
	for _, ms := range []string{"440", "550"} {
		if !strings.Contains(preview[0], `<break time="`+ms+`ms"/>`) {
			t.Errorf("part conversion missing %sms break: %s", ms, preview[0])
		}
	}

	// Two 100ms chunks around 2.5s of 48kHz 16-bit mono silence
	const minBytes = 48000 * 2 * 27 / 10
	if rec.Body.Len() < minBytes {
		t.Errorf("test audio has %d bytes, want at least %d", rec.Body.Len(), minBytes)
	}
	info, err := os.Stat(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if int(info.Size()) != rec.Body.Len() {
		t.Errorf("chapter audio %d bytes, test audio %d bytes", info.Size(), rec.Body.Len())
	}
}

func TestZeroPausesDisableBreaks(t *testing.T) {
	s, _, fake := newSynthesisServer(t)
	zero := map[string]any{}
	for key := range distinctPauses {
		zero[key] = 0
	}
	postJSON(t, s.saveSettings, zero)

	testVoice(t, s, pauseText)
	if got := strings.Join(fake.sent(), "|"); strings.Contains(got, "<break") {
		t.Errorf("expected no breaks, got %s", got)
	}
}

func TestPauseSettingsPersist(t *testing.T) {
	s, db, _ := newSynthesisServer(t)
	postJSON(t, s.saveSettings, distinctPauses)

	for name, cfg := range map[string]*config.Config{"running": s.worker.cfg, "restarted": loadSavedConfig(t, db)} {
		got := []float64{
			float64(cfg.SentenceBreakMs), float64(cfg.ParagraphBreakMs), float64(cfg.DashBreakDurationMs),
			float64(cfg.ParenthesesBreakMs), float64(cfg.ColonBreakMs), float64(cfg.TitleBreakMs),
			cfg.ChapterGapSeconds, cfg.PartGapSeconds,
		}
		want := []float64{110, 220, 330, 440, 550, 660, 1.5, 2.5}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s config: setting %d = %v, want %v", name, i, got[i], want[i])
			}
		}
	}
}
