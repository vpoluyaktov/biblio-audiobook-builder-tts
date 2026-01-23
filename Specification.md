# Biblio Audiobook Builder TTS - Specifications

> Part of the [BiblioHub](https://github.com/vpoluyaktov/BiblioHub) application suite

## Overview

**Biblio Audiobook Builder TTS** is a server-based application that converts eBooks (EPUB, FB2) into audiobooks using various Text-to-Speech engines. It provides a web interface for uploading books, monitoring conversion progress, and downloading completed audiobooks.

### Key Features

- **Server-Client Architecture**: Offloads CPU-intensive TTS processing to a remote server
- **Drop-and-Forget Model**: Start a conversion, close the browser, reconnect later to check progress
- **Multi-Client Support**: Multiple users can upload, monitor, and download jobs simultaneously
- **Multiple TTS Engines**: Supports local (espeak, festival) and cloud-based (Google, Azure) TTS providers
- **Real-time Progress**: WebSocket-based live updates on conversion status
- **Chunked Processing**: Automatically splits text into sentences/paragraphs for reliable TTS conversion

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web Browser (Client)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ Upload Form │  │  Jobs List  │  │  WebSocket Connection   │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP / WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Server                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  REST API   │  │Static Assets│  │    WebSocket Hub        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Job Management                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Job Store  │  │   Worker    │  │    Job Queue            │  │
│  │ (In-Memory) │  │ (Goroutine) │  │                         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        TTS Service                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Adapter   │  │   Chunker   │  │      Providers          │  │
│  │ (Chunking)  │  │ (Sentence)  │  │ (espeak/google/azure)   │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Book Parsers                              │
│  ┌─────────────────────────┐  ┌─────────────────────────────┐   │
│  │      EPUB Parser        │  │       FB2 Parser            │   │
│  └─────────────────────────┘  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Package Structure

```
biblio-audiobook-builder-tts/
├── main.go                     # Application entry point
├── biblio-audiobook-builder-tts.config.yaml  # Configuration file
├── go.mod                      # Go module definition
├── Specifications.md           # This file
│
├── internal/
│   ├── config/
│   │   └── config.go           # Configuration loading (viper)
│   │
│   ├── server/
│   │   ├── server.go           # HTTP server, REST API endpoints
│   │   ├── job.go              # Job model and JobStore
│   │   ├── worker.go           # Background job processor
│   │   ├── websocket.go        # WebSocket hub for real-time updates
│   │   ├── assets/
│   │   │   ├── style.css       # Web UI styles
│   │   │   └── app.js          # Frontend JavaScript
│   │   └── templates/
│   │       └── index.html      # Main web page template
│   │
│   ├── tts/
│   │   ├── service.go          # TTS service interface
│   │   ├── provider.go         # Provider interface
│   │   ├── adapter.go          # Chunking adapter wrapper
│   │   ├── chunker.go          # Text chunking (sentence/paragraph)
│   │   ├── local_provider.go   # Local TTS (espeak, festival)
│   │   └── cloud_provider.go   # Cloud TTS (Google, Azure)
│   │
│   ├── parser/
│   │   ├── parser.go           # Parser interface
│   │   ├── epub_parser.go      # EPUB format parser
│   │   └── fb2_parser.go       # FB2 format parser
│   │
│   ├── utils/
│   │   └── process.go          # Process management utilities
│   │
│   ├── dto/                    # Data Transfer Objects
│   ├── mq/                     # Message Queue (legacy TUI)
│   ├── controller/             # Controllers (legacy TUI)
│   ├── ui/                     # TUI interface (legacy)
│   ├── app/                    # App state (legacy)
│   └── monitoring/             # Metrics collection
│
└── output/                     # Generated audiobooks
```

---

## REST API

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/` | Main web interface |
| `GET` | `/assets/*` | Static assets (CSS, JS) |
| `GET` | `/api/jobs` | List all jobs |
| `GET` | `/api/jobs/{id}` | Get job details |
| `DELETE` | `/api/jobs/{id}` | Delete/cancel a job |
| `GET` | `/api/jobs/{id}/download` | Get download info for completed job |
| `POST` | `/api/upload` | Upload book and create conversion job |
| `GET` | `/api/providers` | List available TTS providers |
| `GET` | `/api/voices` | List available voices for a provider |
| `GET` | `/api/config` | Get client-relevant configuration |
| `WS` | `/api/ws` | WebSocket for real-time updates |
| `GET` | `/api/opds/sources` | List OPDS catalog sources |
| `POST` | `/api/opds/sources` | Add new OPDS source |
| `GET` | `/api/opds/sources/{id}` | Get OPDS source details |
| `PUT` | `/api/opds/sources/{id}` | Update OPDS source |
| `DELETE` | `/api/opds/sources/{id}` | Delete OPDS source |
| `GET` | `/api/opds/browse?url=` | Browse OPDS catalog at URL |
| `GET` | `/api/opds/search?url=&q=` | Search OPDS catalog |
| `POST` | `/api/opds/download` | Download book from OPDS and create preview |
| `POST` | `/api/opds/convert` | Start conversion from OPDS preview |
| `GET` | `/api/opds/proxy?url=` | Proxy requests to OPDS (for CORS) |

### WebSocket Messages

**Server → Client:**
```json
{
  "type": "job_created|job_updated|job_progress|job_completed|job_failed|job_deleted",
  "payload": { /* job object */ }
}
```

### Job Object

```json
{
  "id": "uuid",
  "file_name": "book.epub",
  "file_path": "/path/to/uploaded/file",
  "status": "pending|parsing|converting|completed|failed|cancelled",
  "progress": 0.75,
  "current_chapter": "Chapter 5",
  "current_chapter_num": 5,
  "total_chapters": 20,
  "book_title": "Pride and Prejudice",
  "book_author": "Jane Austen",
  "provider": "espeak",
  "voice": "en",
  "speed": 1.0,
  "pitch": 1.0,
  "output_path": "/output/Pride and Prejudice",
  "error": "",
  "created_at": "2026-01-12T08:00:00Z",
  "updated_at": "2026-01-12T08:30:00Z"
}
```

---

## Configuration

### abb_tts.config.yaml

```yaml
# Basic settings
log_file: "abb_tts.log"
output_dir: "./output"
temp_dir: "./temp"
default_voice: "en-US"
default_provider: "espeak"

# Server settings
server_port: "8080"
server_host: "0.0.0.0"
open_browser: true

# TTS settings
bit_rate_kbs: 128
sample_rate_hz: 44100
default_speed: 1.0
default_pitch: 1.0

# Cloud provider settings (optional)
cloud_api_key: ""
google_tts_endpoint: ""
azure_tts_endpoint: ""

# Audiobookshelf integration (optional)
audiobookshelf_url: ""
audiobookshelf_user: ""
audiobookshelf_password: ""
audiobookshelf_library: ""
```

### Command Line Flags

```bash
./biblio-audiobook-builder-tts [flags]

Flags:
  --config string      Path to configuration file (default "biblio-audiobook-builder-tts.config.yaml")
  --port string        Port to run the server on (overrides config)
  --host string        Host to bind the server to (overrides config)
  --no-browser         Don't automatically open browser
  --restart            Kill any existing process on the port before starting
  --log-level string   Log level: DEBUG, INFO, WARN, ERROR (default "INFO")
```

---

## Current Development Status

### Completed Features ✅

| Feature | Status | Notes |
|---------|--------|-------|
| HTTP Server | ✅ Done | Serves web UI and REST API |
| Job Management | ✅ Done | Create, list, delete jobs |
| Job Store | ✅ Done | In-memory storage with thread safety |
| WebSocket Hub | ✅ Done | Real-time updates to multiple clients |
| Background Worker | ✅ Done | Processes jobs asynchronously |
| File Upload | ✅ Done | EPUB and FB2 support |
| TTS Service | ✅ Done | Provider abstraction |
| Text Chunking | ✅ Done | Sentence/paragraph splitting |
| Local TTS (espeak) | ✅ Done | Via stdin to avoid arg limits |
| Web Frontend | ✅ Done | Modern dark theme UI |
| Restart Flag | ✅ Done | Port-based process detection |
| Log Level Flag | ✅ Done | Configurable verbosity |
| Book Preview | ✅ Done | Metadata, chapters, cover, cost estimate before conversion |
| Cover Image Extraction | ✅ Done | From EPUB/FB2 metadata |
| Server TUI | ✅ Done | Terminal dashboard showing server status, providers, jobs |
| FFmpeg/FFProbe Wrappers | ✅ Done | Ported from abb_ia for audio processing |
| Audiobookshelf Client | ✅ Done | Ported from abb_ia for server integration |
| M4B Builder | ✅ Done | Chapter markers, metadata, cover embedding |
| Worker M4B Integration | ✅ Done | M4B building integrated into worker pipeline |
| Audiobookshelf Integration | ✅ Done | Auto-upload completed books when configured |
| Gap Between Chapters | ✅ Done | Configurable silence padding (chapter_gap_seconds) |
| Pronunciation Dictionary | ✅ Done | Regex-based text fixes with default rules |
| Multi-Part Audiobooks | ✅ Done | Split large books based on max_file_size_mb |
| SQLite Storage Layer | ✅ Done | Config and job persistence with SQLite |
| Settings UI | ✅ Done | Multi-tab configuration modal (General, TTS, Output, Audiobookshelf) |
| OPDS Client | ✅ Done | Browse OPDS catalogs, download books for conversion |
| OpenTTS Integration | ✅ Done | Self-hosted TTS server with multiple engines (Larynx, MaryTTS, NanoTTS, etc.) |
| RHVoice Integration | ✅ Done | Self-hosted TTS server with high-quality voices for Russian, Ukrainian, English, Polish, and other languages |

### In Progress 🔄

| Feature | Status | Notes |
|---------|--------|-------|
| Test Voice UI | 🔄 In Progress | Allow users to test TTS voice before conversion |
| Parallel Processing | 🔄 In Progress | Concurrent TTS encoding and M4B building |
| Configurable SSML Tags | 🔄 In Progress | Toggle paragraph/sentence SSML tags per conversion |

### Not Started ❌

| Feature | Priority | Notes |
|---------|----------|-------|
| Audio Normalization | Low | Consistent volume levels |
| Noise Reduction | Low | Clean up TTS artifacts |
| Cloud TTS (Google) | ✅ Done | Standard, WaveNet, Neural2, Studio voices |
| Cloud TTS (Azure) | Low | Needs API integration |
| Job Persistence | Low | Database storage |

---

## Future Enhancements

### Phase -3: Providers Table Migration (High Priority)

**Goal**: Move all TTS provider configurations from the `config` table to a dedicated `providers` table for better maintainability and extensibility.

#### -3.1 Current State

Currently, TTS provider configurations are stored as individual key-value pairs in the `config` table:
- `google_api_key`, `opentts_url`, `rhvoice_url`, `silero_url`, `openai_api_key`, `azure_tts_key`, `azure_tts_region`
- `tts_workers_<provider>` for per-provider worker counts
- `normalize_<provider>` for per-provider normalization settings

This approach has limitations:
- Hard to add new providers without code changes
- Configuration scattered across multiple keys
- No unified way to enable/disable providers
- Difficult to extend with provider-specific settings

#### -3.2 New Providers Table Schema

```sql
CREATE TABLE IF NOT EXISTS providers (
    id TEXT PRIMARY KEY,           -- Provider identifier (e.g., "espeak", "google", "opentts")
    name TEXT NOT NULL,            -- Display name (e.g., "Google Cloud TTS")
    type TEXT NOT NULL,            -- Provider type: "local", "cloud", "self-hosted"
    enabled BOOLEAN DEFAULT 1,     -- Whether provider is enabled
    
    -- Connection settings
    url TEXT DEFAULT '',           -- Server URL for self-hosted providers
    api_key TEXT DEFAULT '',       -- API key for cloud providers
    region TEXT DEFAULT '',        -- Region for Azure TTS
    
    -- Performance settings
    tts_workers INTEGER DEFAULT 3, -- Number of concurrent TTS workers
    max_chunk_size INTEGER DEFAULT 900, -- Maximum characters per TTS request
    
    -- Processing settings
    normalize_numbers BOOLEAN DEFAULT 1,  -- Enable number normalization
    ssml_support BOOLEAN DEFAULT 0,       -- Provider supports SSML markup
    
    -- Metadata
    is_default BOOLEAN DEFAULT 0,  -- Default provider for new jobs
    display_order INTEGER DEFAULT 0, -- Order in UI dropdowns
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_providers_enabled ON providers(enabled);
CREATE INDEX IF NOT EXISTS idx_providers_type ON providers(type);
```

#### -3.3 Provider Struct

```go
// TTSProvider represents a TTS provider configuration stored in the database
type TTSProvider struct {
    ID               string    `json:"id"`
    Name             string    `json:"name"`
    Type             string    `json:"type"`  // "local", "cloud", "self-hosted"
    Enabled          bool      `json:"enabled"`
    URL              string    `json:"url"`      // Server URL for self-hosted providers
    APIKey           string    `json:"api_key"`  // API key for cloud providers
    Region           string    `json:"region"`   // Region for Azure TTS
    TTSWorkers       int       `json:"tts_workers"`
    MaxChunkSize     int       `json:"max_chunk_size"` // Maximum characters per TTS request
    NormalizeNumbers bool      `json:"normalize_numbers"`
    SSMLSupport      bool      `json:"ssml_support"`  // Provider supports SSML markup
    IsDefault        bool      `json:"is_default"`
    DisplayOrder     int       `json:"display_order"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

#### -3.4 Default Providers

On first run, initialize with default providers:

| ID | Name | Type | Enabled | SSML | Max Chunk | URL | API Key |
|----|------|------|---------|------|-----------|-----|---------|
| espeak | eSpeak | local | true | false | 5000 | - | - |
| google | Google Cloud TTS | cloud | false | true | 4000 | - | (required) |
| openai | OpenAI TTS | cloud | false | false | 4000 | - | (required) |
| azure | Azure TTS | cloud | false | true | 4000 | - | (required + region) |
| opentts | OpenTTS | self-hosted | false | false | 2000 | (required) | - |
| rhvoice | RHVoice | self-hosted | false | true | 2000 | (required) | - |
| silero | Silero TTS | self-hosted | false | true | 900 | (required) | - |

#### -3.5 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/providers` | List all providers (with status) |
| `GET` | `/api/providers/{id}` | Get provider details |
| `PUT` | `/api/providers/{id}` | Update provider configuration |
| `POST` | `/api/providers/{id}/test` | Test provider connection |
| `POST` | `/api/providers/{id}/enable` | Enable provider |
| `POST` | `/api/providers/{id}/disable` | Disable provider |

#### -3.6 Migration Strategy

1. Create new `providers` table
2. Migrate existing config values to providers table
3. Update TTS service to load providers from database
4. Update settings API to use providers endpoints
5. Update UI to use new providers management
6. Remove old config keys (backward compatibility period)

#### -3.7 UI Changes

**Settings Modal - TTS Providers Tab:**
```
┌─────────────────────────────────────────────────────────────────┐
│  🔊 TTS Providers                                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ☑ eSpeak (Local)                              [Default] │   │
│  │   Status: ✅ Available                                  │   │
│  │   Workers: [3 ▼]  Normalize: [✓]                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ☐ Google Cloud TTS (Cloud)                              │   │
│  │   API Key: [••••••••••••••••••••] [Test]               │   │
│  │   Workers: [3 ▼]  Normalize: [  ]                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ☑ OpenTTS (Self-hosted)                                 │   │
│  │   URL: [http://localhost:5500    ] [Test]               │   │
│  │   Status: ✅ Connected (45 voices)                      │   │
│  │   Workers: [3 ▼]  Normalize: [✓]                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### -3.8 Implementation Tasks

- [x] Add `providers` table schema to db.go migrate()
- [x] Create `Provider` struct in storage/db.go
- [x] Implement CRUD operations: CreateProvider, GetProvider, UpdateProvider, ListProviders
- [x] Add InitializeDefaultProviders() function
- [x] Add migration logic to move config values to providers table
- [ ] Update config.Config to remove provider-specific fields (deferred - backward compatibility)
- [x] Update TTS service to load providers from database
- [x] Add `/api/providers` endpoints in server.go
- [x] Add `/api/providers/{id}/test` endpoint
- [ ] Update settings.go to use providers API (existing settings still work via migration)
- [ ] Update Settings UI with new Providers tab (future enhancement)
- [x] Update main upload form provider dropdown
- [x] Add provider status indicators to UI
- [x] Test migration from existing config
- [ ] Update TUI to show provider status from database (future enhancement)

#### -3.9 Per-Provider Chunk Size (Branch: `feature/per-provider-chunk-size`)

**Goal**: Allow each TTS provider to have its own maximum chunk size limit, optimizing quality for providers with higher limits while ensuring compatibility with providers like Silero that have strict 1000-char limits.

**Background**: Different TTS providers have different text length limits:
- Silero: 1000 characters (hard limit from model)
- Google Cloud TTS: 5000 bytes (~4000 chars for ASCII, less for multi-byte)
- OpenAI TTS: 4096 characters
- Azure TTS: No hard limit (billed per character)
- eSpeak/RHVoice/OpenTTS: No strict limits

**Implementation Tasks**:
- [x] Add `max_chunk_size` column to providers table in db.go migrate()
- [x] Update TTSProvider struct with MaxChunkSize field
- [x] Update InitializeDefaultProviders() with per-provider chunk sizes
- [x] Modify GetAdapter() in service.go to pass provider's MaxChunkSize to chunker
- [x] Update NewAdapter() to accept custom ChunkerConfig (already supported)
- [x] Add migration to set max_chunk_size for existing providers
- [x] Update Settings UI to show/edit max_chunk_size per provider
- [ ] Test with Silero (900), Google (4000), and other providers

---

### Phase -2: Parallel Processing (High Priority)

**Goal**: Improve conversion speed by processing multiple chapters and M4B parts concurrently using configurable worker pools.

#### -2.1 Feature Overview

Currently, the application processes chapters sequentially:
1. Chapter 1 → TTS → Save WAV
2. Chapter 2 → TTS → Save WAV
3. ...
4. Build M4B Part 1
5. Build M4B Part 2
6. ...

With parallel processing:
1. Chapters 1-N processed concurrently (N = TTS workers)
2. M4B Parts 1-M built concurrently (M = encoder workers)

#### -2.2 Configuration Settings

New configuration fields:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `concurrent_tts_workers` | int | 3 | Number of parallel TTS conversion workers |
| `concurrent_encoders` | int | 2 | Number of parallel M4B encoding workers |

**Per-Provider TTS Workers:**

Different TTS providers have different rate limits and resource requirements:
- **Local (espeak, festival)**: Limited by CPU cores
- **Self-hosted (OpenTTS, RHVoice)**: Limited by server capacity
- **Cloud (Google, Azure, OpenAI)**: Limited by API rate limits

#### -2.3 JobDispatcher Utility

Port the `JobDispatcher` from abb_ia project to manage worker pools:

```go
// internal/utils/job_dispatcher.go
package utils

type JobDispatcher struct {
    workers []worker
    jobs    []job
    stopCh  chan struct{}
}

type worker struct {
    id   int
    busy bool
    job  *job
}

type job struct {
    id       int
    jobFn    interface{}
    params   []interface{}
    assigned bool
    complete bool
}

func NewJobDispatcher(numWorkers int) *JobDispatcher
func (d *JobDispatcher) AddJob(id int, jobFn interface{}, params ...interface{})
func (d *JobDispatcher) Start()
func (d *JobDispatcher) Stop()
func (d *JobDispatcher) IsComplete(jobId int) bool
func (d *JobDispatcher) GetProgress() (completed int, total int)
```

#### -2.4 Parallel TTS Conversion

```go
// Worker.convertBook() changes:
func (w *Worker) convertBook(job *Job, book *parser.Book) (string, []string, error) {
    // Initialize progress tracking for all chapters
    chapterProgress := make([]ChapterProgress, len(book.Chapters))
    
    // Create job dispatcher with configured workers
    jd := utils.NewJobDispatcher(w.cfg.ConcurrentTTSWorkers)
    
    // Add all chapters as jobs
    for i, chapter := range book.Chapters {
        jd.AddJob(i, w.convertChapter, job, i, chapter, &chapterProgress[i])
    }
    
    // Start progress update goroutine
    go w.updateChapterProgress(job, chapterProgress)
    
    // Process all chapters in parallel
    jd.Start()
    
    // Collect results in order
    return w.collectChapterFiles(chapterProgress)
}
```

#### -2.5 Parallel M4B Building

```go
// Worker.buildM4B() changes:
func (w *Worker) buildM4B(job *Job, book *parser.Book, chapterFiles []string) (string, error) {
    parts, _ := audio.SplitIntoParts(chapterFiles, chapterTitles, w.cfg.MaxFileSizeMB)
    
    // Initialize progress tracking for all parts
    partProgress := make([]PartProgress, len(parts))
    
    // Create job dispatcher for encoding
    jd := utils.NewJobDispatcher(w.cfg.ConcurrentEncoders)
    
    // Add all parts as jobs
    for i, part := range parts {
        jd.AddJob(i, w.buildPart, job, i, part, &partProgress[i])
    }
    
    // Start progress update goroutine
    go w.updatePartProgress(job, partProgress)
    
    // Build all parts in parallel
    jd.Start()
    
    return w.collectM4BFiles(partProgress)
}
```

#### -2.6 Progress Tracking

**ChapterProgress struct:**
```go
type ChapterProgress struct {
    ChapterNum   int
    ChapterTitle string
    Status       string  // "pending", "converting", "complete", "error"
    Progress     float64 // 0.0 - 1.0
    OutputFile   string
    Error        string
}
```

**PartProgress struct:**
```go
type PartProgress struct {
    PartNum      int
    Status       string  // "pending", "encoding", "complete", "error"
    Progress     float64 // 0.0 - 1.0
    OutputFile   string
    Error        string
}
```

#### -2.7 UI Changes

**Progress Display Updates:**

Current UI shows single progress bar. New UI shows:

```
Converting: Pride and Prejudice
├── Chapter Progress: 45/61 (73%)
│   ├── Worker 1: Chapter 46 - "Elizabeth's Dilemma" [████░░░░░░] 42%
│   ├── Worker 2: Chapter 47 - "Mr. Darcy Returns" [██████░░░░] 65%
│   └── Worker 3: Chapter 48 - "The Letter" [████████░░] 81%
│
└── Overall: [████████████████░░░░░░░░░░░░░░] 73%
```

**Building M4B:**
```
Building M4B: Pride and Prejudice
├── Part Progress: 1/3 complete
│   ├── Encoder 1: Part 2 [████████░░░░░░░░░░░░] 40%
│   └── Encoder 2: Part 3 [██░░░░░░░░░░░░░░░░░░] 10%
│
└── Overall: [██████████░░░░░░░░░░░░░░░░░░░░] 33%
```

**WebSocket Message Updates:**

New message type for parallel progress:
```json
{
  "type": "job_parallel_progress",
  "payload": {
    "job_id": "uuid",
    "phase": "converting",  // "converting" or "building"
    "workers": [
      {
        "worker_id": 0,
        "item_num": 46,
        "item_title": "Elizabeth's Dilemma",
        "progress": 0.42,
        "status": "active"
      },
      {
        "worker_id": 1,
        "item_num": 47,
        "item_title": "Mr. Darcy Returns",
        "progress": 0.65,
        "status": "active"
      }
    ],
    "completed": 45,
    "total": 61,
    "overall_progress": 0.73
  }
}
```

#### -2.8 Settings UI Changes

Add new "Performance" section to Settings modal:

**Performance Tab:**
```
┌─────────────────────────────────────────────────────────┐
│  ⚡ Performance Settings                                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  TTS Conversion Workers                                 │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Concurrent TTS Workers: [3    ] ▼              │   │
│  │  (1-10, higher = faster but more CPU/memory)   │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  M4B Encoding Workers                                   │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Concurrent Encoders: [2    ] ▼                 │   │
│  │  (1-5, higher = faster but more CPU/disk I/O)  │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ℹ️ Note: Cloud TTS providers may have rate limits.    │
│     Reduce workers if you encounter API errors.        │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### -2.9 Implementation Tasks

- [ ] Port `JobDispatcher` from abb_ia to `internal/utils/job_dispatcher.go`
- [ ] Add `concurrent_tts_workers` and `concurrent_encoders` to config
- [ ] Add config fields to `LoadFromDB` function
- [ ] Add Performance tab to Settings UI
- [ ] Create `ChapterProgress` and `PartProgress` structs
- [ ] Refactor `convertBook` for parallel chapter processing
- [ ] Refactor `buildM4B` for parallel part encoding
- [ ] Add progress tracking goroutines
- [ ] Update WebSocket messages for parallel progress
- [ ] Update frontend to display multi-worker progress
- [ ] Add unit tests for JobDispatcher
- [ ] Test with various worker counts and book sizes

---

### Phase -1: Test Voice UI (High Priority)

**Goal**: Allow users to test TTS voices before starting a book conversion by generating a short audio sample with custom text.

#### -1.1 Feature Overview

User workflow:
1. Upload an ebook or select from OPDS catalog
2. Select provider, language, model, and voice on the main screen
3. Adjust speed and pitch settings
4. Click "Test Voice" button to open test dialog
5. In the dialog, enter custom test text and click "Generate Audio"
6. Listen to the sample directly in the browser
7. Close dialog and adjust settings if needed (can test again)
8. Proceed with book preview or conversion

#### -1.2 API Endpoint

**New Endpoint:**
```
POST /api/test-voice
```

**Request:**
```json
{
  "provider": "espeak",
  "voice": "en-us",
  "text": "Hello, this is a test of the text to speech voice.",
  "speed": 1.0,
  "pitch": 1.0
}
```

**Response:**
- Content-Type: `audio/wav` or `audio/mpeg`
- Body: Raw audio data

**Error Response:**
```json
{
  "error": "TTS provider not available"
}
```

#### -1.3 UI Component

The test voice feature is accessible from the **main upload form** via a "Test Voice" button that opens a modal dialog.

**Test Voice Dialog contains:**
- Display of currently selected provider, voice, speed, and pitch
- Text input field (default: "Hello, this is a test of the text to speech voice.")
- "Generate Audio" button
- Audio player (HTML5 `<audio>` element)
- Loading indicator during generation
- Status message (success/error)

#### -1.4 Implementation Tasks

- [x] Add `POST /api/test-voice` endpoint in server.go
- [x] Create TTS test function that generates short audio sample
- [x] Add "Test Voice" button to main upload form
- [x] Create test voice modal dialog
- [x] Implement generateTestVoice() in app.js
- [x] Limit test text length (max 500 characters)
- [x] Add loading state and error handling
- [x] Add CSS styles for test voice modal

---

### Phase -0.5: RHVoice Integration (High Priority)

**Goal**: Integrate RHVoice REST server as a TTS provider for high-quality speech synthesis in Russian, Ukrainian, English, Polish, and other languages.

#### -0.5.1 RHVoice REST API

RHVoice is a free and open-source speech synthesizer supporting multiple languages with natural-sounding voices.

**Server Endpoints:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/info` | GET | Server info, supported voices, formats |
| `/voices` | GET | List available voices grouped by language |
| `/say?text=...&voice=...` | GET | Synthesize speech (text in URL) |
| `/rhasspy?voice=...` | POST | Synthesize speech (text in body, WAV output) |

**Query Parameters:**

| Parameter | Description | Default | Range |
|-----------|-------------|---------|-------|
| `voice` | Voice name (e.g., alan, anna, aleksandr) | anna | See /voices |
| `format` | Output format (wav, mp3, opus, flac) | mp3 | GET only |
| `rate` | Speech rate | 50 | 0-100 |
| `pitch` | Voice pitch | 50 | 0-100 |
| `volume` | Voice volume | 50 | 0-100 |

**POST Method (Recommended):**
- Use `/rhasspy` endpoint for long text (avoids URL length limits)
- Text sent in request body (plain text)
- Always returns WAV format
- Voice and other params via query string

**Example:**
```bash
curl -X POST --data "Hello world" "http://server:8080/rhasspy?voice=alan&rate=50"
```

#### -0.5.2 Supported Languages & Voices

| Language | Voices |
|----------|--------|
| English (US) | alan, bdl, clb, evgeniy-eng, lyubov, slt |
| Russian | aleksandr, aleksandr-hq, anna, arina, artemiy, elena, evgeniy-rus, irina, mikhail, pavel, tatiana, timofey, umka, victoria, vitaliy, vitaliy-ng, vsevolod, yuriy |
| Ukrainian | anatol, marianna, natalia, volodymyr |
| Polish | alicja, cezary, magda, michal, natan |
| And more | Czech, Slovak, Georgian, Kyrgyz, Macedonian, etc. |

#### -0.5.3 Implementation Tasks

- [x] Add `rhvoice_url` config field in config.go
- [x] Add `rhvoice_url` to LoadFromDB in config.go
- [x] Create `rhvoice_provider.go` with RHVoiceProvider struct
- [x] Implement `/info` endpoint parsing for voices
- [x] Implement `ConvertToSpeech` using POST to `/rhasspy`
- [x] Implement `GetAvailableVoices`, `GetAvailableLanguages`
- [x] Register provider in service.go
- [x] Add RHVoice URL field to Settings UI (TTS tab)
- [x] Add test connection button for RHVoice in Settings
- [x] Fix WAV concatenation for RHVoice streaming format
- [x] Fix test voice Unicode character limit
- [ ] Add unit tests for RHVoice provider
- [ ] Update README with RHVoice configuration

---

### Phase -0.25: Text Normalization (High Priority)

**Goal**: Preprocess text to convert numbers to words before TTS processing. Many TTS models don't handle numbers well, so converting "Chapter 5" to "Chapter five" or "глава 5" to "глава пятая" improves audio quality.

#### -0.25.1 Feature Overview

Text normalization converts numeric values to their word equivalents based on:
- **Language**: Different languages have different number words
- **Form**: Cardinal (one, two, three) vs Ordinal (first, second, third)
- **Gender**: Required for languages like Russian (один/одна/одно)
- **Case**: Required for languages like Russian (nominative, genitive, etc.)

**Challenges for Russian:**
- Gender agreement: "глава первая" (feminine) vs "том первый" (masculine)
- Cardinal vs Ordinal: "2 рубля" (cardinal) vs "глава 2" → "глава вторая" (ordinal)
- Case agreement: Different word endings based on grammatical case

#### -0.25.2 Package Structure

```
internal/normalize/
├── types.go           # Core types: Gender, Form, Case, Context, NumberConverter interface
├── registry.go        # Language converter registry
├── english.go         # English number-to-words implementation
├── russian.go         # Russian number-to-words implementation (with gender/case)
├── processor.go       # Text processor: finds numbers and replaces with words
├── nouns.go           # Noun database for context detection (gender, ordinal triggers)
├── english_test.go    # English converter tests
├── russian_test.go    # Russian converter tests
└── processor_test.go  # Text processor tests
```

#### -0.25.3 Core Types

```go
// Gender represents grammatical gender
type Gender int
const (
    Masculine Gender = iota
    Feminine
    Neuter
)

// Form represents cardinal or ordinal
type Form int
const (
    Cardinal Form = iota  // one, two, three
    Ordinal               // first, second, third
)

// Case represents grammatical case
type Case int
const (
    Nominative Case = iota
    Genitive
    Dative
    Accusative
    Instrumental
    Prepositional
)

// Context provides grammatical context for number conversion
type Context struct {
    Form   Form
    Gender Gender
    Case   Case
}

// NumberConverter interface for language-specific implementations
type NumberConverter interface {
    ToWords(n int64, ctx Context) string
    SupportsContext() bool
    LanguageCode() string
    LanguageName() string
}
```

#### -0.25.4 English Implementation

English is simpler - no gender/case, just cardinal/ordinal:

| Number | Cardinal | Ordinal |
|--------|----------|---------|
| 1 | one | first |
| 2 | two | second |
| 3 | three | third |
| 21 | twenty-one | twenty-first |
| 100 | one hundred | one hundredth |

**Coverage**: 0 to 999,999,999,999 (trillions)

#### -0.25.5 Russian Implementation

Russian requires gender and form awareness:

| Number | Masculine Cardinal | Feminine Cardinal | Masculine Ordinal | Feminine Ordinal |
|--------|-------------------|-------------------|-------------------|------------------|
| 1 | один | одна | первый | первая |
| 2 | два | две | второй | вторая |
| 3 | три | три | третий | третья |
| 21 | двадцать один | двадцать одна | двадцать первый | двадцать первая |

**Ordinal Trigger Nouns** (use ordinal form):
- глава (chapter) - feminine
- страница (page) - feminine
- часть (part) - feminine
- раздел (section) - masculine
- том (volume) - masculine
- книга (book) - feminine

**Cardinal Nouns** (use cardinal form):
- рубль (ruble) - masculine
- копейка (kopeck) - feminine
- год (year) - masculine
- день (day) - masculine

#### -0.25.6 Text Processor

The processor scans text for numbers and replaces them based on context:

```go
type Processor struct {
    converters map[string]NumberConverter
    nounDB     *NounDatabase
}

// Process replaces numbers in text with words
func (p *Processor) Process(text, lang string) string

// Example:
// Input:  "Chapter 5 contains 42 pages"
// Output: "Chapter five contains forty-two pages"

// Russian example:
// Input:  "Глава 5 содержит 42 страницы"
// Output: "Глава пятая содержит сорок две страницы"
```

**Context Detection Algorithm:**
1. Find number in text using regex `\d+`
2. Look at surrounding words (before and after)
3. Check noun database for known nouns
4. Determine form (cardinal/ordinal) and gender from noun
5. Convert number using appropriate context
6. Replace in text

#### -0.25.7 Noun Database

```go
type NounInfo struct {
    Gender       Gender
    TriggerForm  Form   // Cardinal or Ordinal
    SingularForm string
    PluralForms  []string
}

var russianNouns = map[string]NounInfo{
    "глава":    {Feminine, Ordinal, "глава", []string{"главы", "глав"}},
    "страница": {Feminine, Ordinal, "страница", []string{"страницы", "страниц"}},
    "часть":    {Feminine, Ordinal, "часть", []string{"части", "частей"}},
    "том":      {Masculine, Ordinal, "том", []string{"тома", "томов"}},
    "рубль":    {Masculine, Cardinal, "рубль", []string{"рубля", "рублей"}},
    // ... more nouns
}
```

#### -0.25.8 Integration with TTS Pipeline

The text processor will be called before text is sent to TTS:

```go
// In tts/adapter.go or tts/service.go
func (s *Service) ConvertToSpeech(text, lang string) ([]byte, error) {
    // Normalize text (convert numbers to words)
    normalizedText := s.normalizer.Process(text, lang)
    
    // Send to TTS provider
    return s.provider.Synthesize(normalizedText)
}
```

#### -0.25.9 Configuration

Per-provider normalization settings in config:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `normalize_espeak` | bool | true | Enable normalization for espeak (reads digits, needs it) |
| `normalize_rhvoice` | bool | true | Enable normalization for RHVoice |
| `normalize_silero` | bool | true | Enable normalization for Silero |
| `normalize_opentts` | bool | true | Enable normalization for OpenTTS |
| `normalize_google` | bool | false | Disable for Google (handles numbers well) |
| `normalize_openai` | bool | false | Disable for OpenAI (handles numbers well) |
| `normalize_azure` | bool | false | Disable for Azure (handles numbers well) |

**Default behavior by provider:**
- **espeak, rhvoice, silero, opentts**: Normalization ON (these providers read numbers as digits)
- **google, openai, azure**: Normalization OFF (these providers handle numbers natively)

**Config method:**
```go
// NeedsNormalization returns whether text normalization should be applied
func (c *Config) NeedsNormalization(provider string) bool
```

#### -0.25.10 Implementation Tasks

- [x] Create `internal/normalize/types.go` with core types and interfaces
- [x] Create `internal/normalize/english.go` with English converter
- [x] Create `internal/normalize/english_test.go` with tests
- [x] Create `internal/normalize/russian.go` with Russian converter
- [x] Create `internal/normalize/russian_test.go` with tests
- [x] Create `internal/normalize/nouns.go` with noun database
- [x] Create `internal/normalize/processor.go` with text processor
- [x] Create `internal/normalize/processor_test.go` with tests
- [x] Add per-provider `normalize_<provider>` config fields
- [x] Add `NeedsNormalization(provider)` config method with sensible defaults
- [x] Integrate processor into TTS pipeline (worker.go)
- [ ] Add normalization toggle to Settings UI (per provider)

---

### Phase -0.24: SSML Text Processing

**Goal**: Wrap text in SSML (Speech Synthesis Markup Language) tags when sending to TTS providers that support SSML, improving speech quality with proper paragraph and sentence pauses.

#### -0.24.1 SSML Structure

When `ssml_support` is enabled for a provider, text is wrapped with SSML tags:

```xml
<speak>
<p>
    <s>First sentence of the paragraph.</s>
    <s>Second sentence of the paragraph.</s>
</p>
<p>
    <s>First sentence of next paragraph.</s>
    <s>Another sentence here.</s>
</p>
</speak>
```

**Tags used:**
- `<speak>` - Root element required by SSML
- `<p>` - Paragraph tag, adds natural pause between paragraphs
- `<s>` - Sentence tag, adds natural pause between sentences

#### -0.24.2 Text Processing Logic

1. **Paragraph Detection**: Split text by double newlines (`\n\n`) or single newlines
2. **Sentence Detection**: Split paragraphs by sentence-ending punctuation (`. ! ? ։ ؟` etc.)
3. **XML Escaping**: Escape special characters (`& < > " '`) before wrapping
4. **Empty Content Handling**: Skip empty paragraphs and sentences

```go
// WrapTextInSSML converts plain text to SSML format
func WrapTextInSSML(text string) string {
    // Split into paragraphs
    paragraphs := splitIntoParagraphs(text)
    
    var result strings.Builder
    result.WriteString("<speak>\n")
    
    for _, para := range paragraphs {
        if strings.TrimSpace(para) == "" {
            continue
        }
        result.WriteString("<p>\n")
        
        // Split paragraph into sentences
        sentences := splitIntoSentences(para)
        for _, sent := range sentences {
            sent = strings.TrimSpace(sent)
            if sent == "" {
                continue
            }
            result.WriteString("    <s>")
            result.WriteString(escapeXML(sent))
            result.WriteString("</s>\n")
        }
        result.WriteString("</p>\n")
    }
    
    result.WriteString("</speak>")
    return result.String()
}
```

#### -0.24.3 Sentence Boundary Detection

Sentences are split on:
- Period followed by space or end: `. `
- Exclamation mark: `!`
- Question mark: `?`
- Armenian question mark: `։`
- Arabic question mark: `؟`
- Ellipsis handling: `...` treated as single boundary

**Edge cases handled:**
- Abbreviations (Mr., Dr., etc.) - not split
- Decimal numbers (3.14) - not split
- Quoted speech ending with punctuation
- Multiple punctuation marks (`?!`, `...`)

#### -0.24.4 Provider SSML Support

| Provider | SSML Support | Notes |
|----------|--------------|-------|
| eSpeak | No | Does not support SSML |
| Google Cloud TTS | Yes | Full SSML support |
| OpenAI TTS | No | Plain text only |
| Azure TTS | Yes | Full SSML support |
| OpenTTS | No | Depends on underlying engine |
| RHVoice | Yes | Supports basic SSML |
| Silero | Yes | Supports SSML with prosody tags |

#### -0.24.5 Integration Point

SSML wrapping is applied in the TTS worker before sending text to the provider:

```go
// In internal/tts/worker.go
func (w *Worker) processChapter(text string, provider *storage.TTSProvider) ([]byte, error) {
    processedText := text
    
    // Apply number normalization if enabled
    if provider.NormalizeNumbers {
        processedText = w.normalizer.Process(processedText, w.language)
    }
    
    // Apply SSML wrapping if provider supports it
    if provider.SSMLSupport {
        processedText = ssml.WrapTextInSSML(processedText)
    }
    
    return w.synthesize(processedText)
}
```

#### -0.24.6 Implementation Tasks

- [x] Add `ssml_support` field to providers table schema
- [x] Add `SSMLSupport` field to `TTSProvider` struct
- [x] Update provider CRUD operations to include `ssml_support`
- [x] Add SSML checkbox to provider settings UI
- [x] Create `internal/ssml/ssml.go` with `WrapTextInSSML` function
- [x] Create `internal/ssml/ssml_test.go` with tests
- [x] Integrate SSML processing into TTS worker pipeline
- [ ] Test with Google Cloud TTS and Azure TTS

---

### Phase -0.23: Configurable SSML Paragraph/Sentence Tags (High Priority)

**Goal**: Allow users to toggle SSML paragraph (`<p>`) and sentence (`<s>`) tags on/off per conversion. These tags add natural pauses between paragraphs and sentences, but may not be suitable for all voices or use cases.

#### -0.23.1 Feature Overview

The SSML `<p>` and `<s>` tags are currently applied automatically when a provider has `ssml_support` enabled. However, some voices may produce unnatural pauses or the user may prefer continuous speech without extra pauses.

This feature adds a toggle on the main conversion form (next to speed/pitch controls) that allows users to:
- **Enable**: Apply `<p>` and `<s>` tags for natural paragraph/sentence pauses
- **Disable**: Send plain text without paragraph/sentence markup (only `<speak>` wrapper if SSML is supported)

**Default behavior**: Enabled (current behavior preserved)

#### -0.23.2 UI Changes

Add a checkbox toggle in the upload form, in the same row as speed/pitch controls:

```
┌─────────────────────────────────────────────────────────────────┐
│  Speed: [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 1.0x                  │
│  Pitch: [━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━] 1.0x                  │
│  ☑ Add pauses between paragraphs and sentences                  │
│    (Uses SSML tags for more natural speech rhythm)              │
└─────────────────────────────────────────────────────────────────┘
```

The toggle should:
- Be visible only when the selected provider supports SSML
- Be checked by default
- Save preference to localStorage with other TTS settings
- Be included in the conversion request

#### -0.23.3 API Changes

**Upload/Convert Request:**

Add `use_sentence_pauses` field to the conversion request:

```json
{
  "provider": "google",
  "voice": "en-US-Wavenet-D",
  "speed": 1.0,
  "pitch": 1.0,
  "use_sentence_pauses": true
}
```

**Job Object:**

Add `use_sentence_pauses` field to job storage:

```json
{
  "id": "uuid",
  "use_sentence_pauses": true,
  ...
}
```

#### -0.23.4 Backend Changes

**SSML Wrapping Logic:**

Modify `WrapTextInSSML` or add a new function that respects the toggle:

```go
// WrapTextInSSML wraps text in SSML with optional paragraph/sentence tags
func WrapTextInSSML(text string, useSentencePauses bool) string {
    if !useSentencePauses {
        // Simple wrapper without <p> and <s> tags
        return "<speak>" + escapeXML(text) + "</speak>"
    }
    // Current implementation with <p> and <s> tags
    ...
}
```

**Adapter Changes:**

Pass the `useSentencePauses` option through the TTS pipeline:

```go
type ConvertOptions struct {
    Speed             float64
    Pitch             float64
    SSMLSupport       bool
    UseSentencePauses bool  // New field
}
```

#### -0.23.5 Implementation Tasks

- [x] Add `use_sentence_pauses` field to Job struct in `internal/server/job.go`
- [x] Add `use_sentence_pauses` to job creation in upload handler
- [x] Add `UseSentencePauses` field to `ConvertOptions` in `internal/tts/service.go`
- [x] Modify `WrapTextInSSML` to accept `useSentencePauses` parameter
- [x] Update adapter to pass `useSentencePauses` to SSML wrapping
- [x] Add checkbox to upload form in `index.html` (after pitch control)
- [x] Add JavaScript to show/hide checkbox based on provider SSML support
- [x] Save `useSentencePauses` preference to localStorage
- [x] Add CSS styles for the new checkbox
- [ ] Update test voice modal to respect the setting
- [ ] Test with SSML-supporting providers (Google, RHVoice, Silero)

---

### Phase 0: Book Preview & Cost Estimation (High Priority)

**Goal**: After uploading a book, show a preview with metadata, chapters, and conversion cost estimate before starting the actual conversion.

#### 0.1 Preview Information

| Field | Description |
|-------|-------------|
| **Book Title** | Extracted from EPUB/FB2 metadata |
| **Author** | Extracted from EPUB/FB2 metadata |
| **Description** | Book annotation/summary |
| **Cover Image** | Extracted and displayed in UI |
| **Chapter List** | All chapters with titles and TOC depth |
| **Total Chapters** | Count of chapters |
| **Character Count** | Total text characters across all chapters |
| **Word Count** | Total words across all chapters |
| **Estimated Duration** | Based on average speech rate (~150 words/min) |
| **Conversion Cost** | $0 for local engines, calculated for cloud TTS |

#### 0.2 Cost Calculation

**Local Engines (espeak, festival):**
- Cost: **$0.00** (free)

**Google Cloud TTS:**
- Standard voices: $4.00 per 1M characters
- WaveNet voices: $16.00 per 1M characters
- Neural2 voices: $16.00 per 1M characters

**Azure Cognitive Services:**
- Neural voices: $16.00 per 1M characters
- Standard voices: $4.00 per 1M characters

**Formula:**
```
cost = (character_count / 1,000,000) * price_per_million
```

#### 0.3 API Changes

**New Endpoint:**
```
POST /api/preview
```

**Request:** `multipart/form-data` with book file

**Response:**
```json
{
  "book_title": "Pride and Prejudice",
  "book_author": "Jane Austen",
  "description": "A romantic novel of manners...",
  "cover_image": "/api/preview/{id}/cover",
  "chapters": [
    {"title": "Chapter 1", "word_count": 2500, "char_count": 15000},
    {"title": "Chapter 2", "word_count": 2200, "char_count": 13200}
  ],
  "total_chapters": 61,
  "total_words": 122000,
  "total_characters": 732000,
  "estimated_duration_minutes": 813,
  "estimated_duration_formatted": "13h 33m",
  "cost_estimates": {
    "espeak": {"cost": 0.00, "currency": "USD", "note": "Free (local)"},
    "festival": {"cost": 0.00, "currency": "USD", "note": "Free (local)"},
    "google_standard": {"cost": 2.93, "currency": "USD", "note": "$4/1M chars"},
    "google_wavenet": {"cost": 11.71, "currency": "USD", "note": "$16/1M chars"},
    "azure_neural": {"cost": 11.71, "currency": "USD", "note": "$16/1M chars"}
  }
}
```

**New Endpoint for Cover Image:**
```
GET /api/preview/{id}/cover
```
Returns the cover image as JPEG/PNG.

#### 0.4 UI Workflow

```
1. User uploads book file
2. Server parses book, extracts metadata
3. Preview page displays:
   - Cover image (if available)
   - Title, Author, Description
   - Chapter list (collapsible)
   - Statistics (words, chars, estimated duration)
   - Cost comparison table for all providers
4. User selects provider, voice, and settings
5. User clicks "Start Conversion" to create job
```

#### 0.5 Implementation Tasks

- [ ] Add `Preview` struct with book metadata and statistics
- [ ] Add character/word counting to parsers
- [ ] Implement cover image extraction (EPUB: from manifest, FB2: from binary)
- [ ] Add `/api/preview` endpoint
- [ ] Add `/api/preview/{id}/cover` endpoint
- [ ] Create preview UI component
- [ ] Add cost calculation logic per provider
- [ ] Store preview temporarily (with TTL cleanup)

---

### Phase 0.5: Server-Side TUI Dashboard (High Priority)

**Goal**: Provide a terminal-based dashboard for server monitoring, similar to `gochess-board` TUI.

#### 0.5.1 TUI Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  🎧  AUDIOBOOK BUILDER TTS SERVER  🎧                                       │
├────────────────────────────────┬────────────────────────────────────────────┤
│  🖥️  SERVER STATUS             │  🔊 TTS PROVIDERS                          │
│                                │                                            │
│  URL:     http://localhost:8080│  # │ Provider │ Type  │ Status │ Voices   │
│  Uptime:  2h 15m 30s           │  1 │ espeak   │ Local │ ✅ OK  │ 12       │
│  Clients: 3 connected          │  2 │ festival │ Local │ ✅ OK  │ 4        │
│                                │  3 │ google   │ Cloud │ ⚠️ No Key│ -       │
│  📡 API ENDPOINTS              │  4 │ azure    │ Cloud │ ⚠️ No Key│ -       │
│  • GET  /api/jobs              │                                            │
│  • POST /api/upload            │                                            │
│  • GET  /api/providers         │                                            │
│  • WS   /api/ws                │                                            │
├────────────────────────────────┴────────────────────────────────────────────┤
│  📋 ACTIVE JOBS                                                             │
│                                                                             │
│  # │ Status     │ Book Title              │ Progress │ Chapter │ Provider  │
│  1 │ Converting │ Pride and Prejudice     │ ████░ 45%│ 12/27   │ espeak    │
│  2 │ Pending    │ War and Peace           │ ░░░░░  0%│ 0/120   │ espeak    │
│  3 │ Completed  │ The Great Gatsby        │ █████100%│ 9/9     │ espeak    │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  Press 'q' to quit │ 'r' to refresh │ 'd' to delete job │ 'c' to cancel    │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 0.5.2 TUI Features

| Feature | Description |
|---------|-------------|
| **Server Status** | URL, uptime, connected WebSocket clients |
| **TTS Providers** | List of configured providers with status |
| **Active Jobs** | Real-time job list with progress bars |
| **Keyboard Controls** | Quit, refresh, delete job, cancel job |
| **Auto-Refresh** | Updates every second |
| **Responsive Layout** | Adapts to terminal size |

#### 0.5.3 Implementation (Bubble Tea)

Using `charmbracelet/bubbletea` framework (same as gochess-board):

```go
package tui

import (
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/table"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    spinner       spinner.Model
    serverURL     string
    startTime     time.Time
    jobStore      *server.JobStore
    ttsService    tts.Service
    wsHub         *server.Hub
    providersTable table.Model
    jobsTable     table.Model
    width, height int
}
```

#### 0.5.4 Command Line Integration

```bash
# Run with TUI (default when terminal is available)
./abb_tts

# Run headless (no TUI, just logs)
./abb_tts --headless

# Run with TUI but don't open browser
./abb_tts --no-browser
```

#### 0.5.5 Implementation Tasks

- [ ] Create `internal/tui/tui.go` with Bubble Tea model
- [ ] Add server status display (URL, uptime, clients)
- [ ] Add TTS providers table with status
- [ ] Add jobs table with real-time progress
- [ ] Implement keyboard controls (q, r, d, c)
- [ ] Add auto-refresh with ticker
- [ ] Integrate with main.go (detect TTY, --headless flag)
- [ ] Add responsive layout for different terminal sizes

---

### Phase 1: M4B Audiobook Output (High Priority)

**Goal**: Generate proper M4B audiobooks with embedded chapters and metadata.

#### 1.1 Audio Processing Pipeline

```
Chapter Audio Files → Re-encode → Add Gaps → Concatenate → Add Chapters → M4B
```

**Implementation:**
- Use ffmpeg for all audio processing
- Re-encode all chapter audio to consistent format (AAC, 128kbps, 44100Hz)
- Generate silence gaps between chapters (configurable, default 2s)
- Create FFMETADATA file with chapter timestamps
- Merge into single M4B with chapter markers

#### 1.2 FFMETADATA Format

```ini
;FFMETADATA1
major_brand=isom
minor_version=1
encoder=abb_tts

[CHAPTER]
TIMEBASE=1/1000
START=0
END=180000
title=Chapter 1

[CHAPTER]
TIMEBASE=1/1000
START=182000
END=360000
title=Chapter 2
```

#### 1.3 FFmpeg Commands

```bash
# Re-encode chapter to AAC
ffmpeg -i chapter_01.wav -ab 128k -ar 44100 -acodec aac chapter_01.aac

# Generate silence gap
ffmpeg -f lavfi -i anullsrc=r=44100:cl=mono -t 2 -ab 128k -ar 44100 gap.aac

# Concatenate with chapters
ffmpeg -f concat -safe 0 -i files.txt -i chapters.txt -map_metadata 1 -c copy output.m4b
```

### Phase 2: Book Metadata & Cover Art

#### 2.1 Cover Image Extraction

**EPUB:**
- Parse `META-INF/container.xml` → `content.opf`
- Find `<meta name="cover" content="cover-id"/>`
- Extract image from manifest

**FB2:**
- Parse `<coverpage><image xlink:href="#cover.jpg"/></coverpage>`
- Extract base64-encoded binary

#### 2.2 Cover Image Embedding

```bash
# Add cover to M4B using AtomicParsley or ffmpeg
AtomicParsley output.m4b --artwork cover.jpg --overWrite

# Or with mutagen (Python) / similar Go library
```

#### 2.3 M4B Metadata Tags

| Tag | Description |
|-----|-------------|
| `©nam` | Title |
| `©alb` | Album (same as title) |
| `©ART` | Artist/Author |
| `©gen` | Genre ("Audiobook") |
| `desc` | Description/Annotation |
| `covr` | Cover artwork |
| `trkn` | Track number (for multi-part) |

### Phase 3: Pronunciation Dictionary

#### 3.1 Dictionary Structure

```
Dict/
├── Default.dict           # Common fixes for all books
├── Voice/
│   ├── espeak.dict        # espeak-specific fixes
│   └── google.dict        # Google TTS fixes
└── Book/
    └── Pride_and_Prejudice.dict  # Book-specific fixes
```

#### 3.2 Dictionary Format

```
# Format: pattern~replacement (regex supported)
Mr\.~Mister
Mrs\.~Missus
Dr\.~Doctor
\bcan't\b~cannot
\bwon't\b~will not
```

#### 3.3 Processing Order

1. Default dictionary
2. Voice-specific dictionary
3. Book-specific dictionary

### Phase 4: Advanced Features

#### 4.1 Multi-Part Audiobooks

For books exceeding size limit (default 2GB):
- Track cumulative size during conversion
- Split at chapter boundaries
- Generate "Book Title, Part 1.m4b", "Part 2.m4b", etc.
- Set track numbers in metadata

#### 4.2 Audio Post-Processing

**Normalization:**
```bash
ffmpeg -i input.aac -filter:a loudnorm output.aac
```

**Noise Reduction:**
- Capture noise sample from TTS silence
- Apply noise reduction filter

#### 4.3 Cloud TTS Integration

**Google Cloud TTS:**
```go
type GoogleTTSRequest struct {
    Input       InputConfig       `json:"input"`
    Voice       VoiceConfig       `json:"voice"`
    AudioConfig AudioConfig       `json:"audioConfig"`
}
```

**Azure Cognitive Services:**
```go
// SSML format for Azure
<speak version="1.0" xmlns="...">
    <voice name="en-US-JennyNeural">
        Text to speak
    </voice>
</speak>
```

#### 4.4 Job Persistence

Replace in-memory JobStore with database:
- SQLite for single-server deployment
- PostgreSQL for multi-server deployment
- Store job state, progress, output paths

#### 4.5 Audiobookshelf Integration

Auto-upload completed audiobooks:
```go
// POST /api/upload
// Headers: Authorization: Bearer <token>
// Body: multipart/form-data with M4B file
```

---

## Dependencies

### Go Modules

```go
require (
    github.com/google/uuid      // Job IDs
    github.com/gorilla/websocket // WebSocket support
    github.com/spf13/viper      // Configuration
)
```

### System Dependencies

| Dependency | Purpose | Required |
|------------|---------|----------|
| `ffmpeg` | Audio processing, M4B creation | Yes |
| `ffprobe` | Audio file inspection | Yes |
| `espeak` | Local TTS engine | Optional |
| `festival` | Local TTS engine | Optional |

### Installation

```bash
# Ubuntu/Debian
sudo apt install ffmpeg espeak

# macOS
brew install ffmpeg espeak

# Build
go build -o abb_tts .

# Run
./abb_tts --no-browser --restart
```

---

## Testing

### Test Structure

```
tests/
├── quality/              # Code quality tests
│   └── test_code.sh      # go fmt, go vet, race detection
├── api/                  # Backend API tests
│   └── test_api.sh       # HTTP endpoint tests
├── integration/          # Integration tests
│   └── test_server.sh    # Full server workflow tests
└── ui/                   # Frontend UI tests (Playwright)
    ├── test_ui.sh
    └── playwright/
        ├── package.json
        ├── playwright.config.js
        └── specs/
            ├── 01-page-load.spec.js
            ├── 02-upload.spec.js
            ├── 03-job-progress.spec.js
            └── 04-download.spec.js
```

### Code Quality Tests

```bash
#!/bin/bash
# tests/quality/test_code.sh

set -e

# Test 1: Code formatting
echo "Testing code formatting..."
unformatted=$(gofmt -l . | grep -v ".git" | grep "\.go$" || true)
if [ -n "$unformatted" ]; then
    echo "Unformatted files: $unformatted"
    exit 1
fi

# Test 2: Static analysis
echo "Running go vet..."
go vet ./...

# Test 3: Race condition detection
echo "Testing for race conditions..."
go test -race -short ./...

# Test 4: Module verification
echo "Verifying modules..."
go mod verify

# Test 5: Unit tests
echo "Running unit tests..."
go test -v ./...

echo "All code quality tests passed!"
```

### Unit Tests

**Test file naming:** `*_test.go` alongside source files

**Example: `internal/tts/chunker_test.go`**
```go
package tts

import "testing"

func TestChunkBySentence(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []string
    }{
        {
            name:     "simple sentences",
            input:    "Hello world. How are you? I am fine!",
            expected: []string{"Hello world.", "How are you?", "I am fine!"},
        },
        {
            name:     "abbreviations",
            input:    "Dr. Smith went to Washington. He met Mr. Jones.",
            expected: []string{"Dr. Smith went to Washington.", "He met Mr. Jones."},
        },
    }

    chunker := NewChunker(DefaultChunkerConfig())
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := chunker.chunkBySentence(tt.input)
            if len(result) != len(tt.expected) {
                t.Errorf("got %d chunks, want %d", len(result), len(tt.expected))
            }
        })
    }
}
```

### API Tests

```bash
#!/bin/bash
# tests/api/test_api.sh

BASE_URL="http://localhost:8080"

# Test providers endpoint
echo "Testing /api/providers..."
curl -s "$BASE_URL/api/providers" | jq .

# Test voices endpoint
echo "Testing /api/voices..."
curl -s "$BASE_URL/api/voices?provider=espeak" | jq .

# Test jobs endpoint
echo "Testing /api/jobs..."
curl -s "$BASE_URL/api/jobs" | jq .

# Test file upload
echo "Testing /api/upload..."
curl -s -X POST -F "file=@test.epub" -F "provider=espeak" \
     "$BASE_URL/api/upload" | jq .
```

### UI Tests (Playwright)

```javascript
// tests/ui/playwright/specs/01-page-load.spec.js
const { test, expect } = require('@playwright/test');

test('page loads correctly', async ({ page }) => {
    await page.goto('http://localhost:8080');
    await expect(page).toHaveTitle(/Audiobook Builder/);
    await expect(page.locator('#upload-form')).toBeVisible();
});

test('job list is visible', async ({ page }) => {
    await page.goto('http://localhost:8080');
    await expect(page.locator('#jobs-list')).toBeVisible();
});
```

### Running Tests

```bash
# All tests
make test

# Code quality only
./tests/quality/test_code.sh

# API tests (requires running server)
./tests/api/test_api.sh

# UI tests (requires running server)
cd tests/ui && ./test_ui.sh

# CI mode (headless)
CI=true ./tests/ui/test_ui.sh
```

---

## CI/CD

### GitHub Actions Workflows

#### Build Workflow (`.github/workflows/build.yaml`)

```yaml
name: Build abb_tts

on:
  push:
    branches: ['*']
  pull_request:
    branches: ['main']

jobs:
  test:
    name: Run Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install system dependencies
        run: |
          sudo apt-get update && sudo apt-get install -y \
            ffmpeg espeak

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run Go tests
        run: go test -v -race ./...

      - name: Run code quality checks
        run: |
          go fmt ./...
          go vet ./...

  build:
    name: Build
    needs: test
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
          - os: macos-latest
            goos: darwin
            goarch: amd64
          - os: macos-latest
            goos: darwin
            goarch: arm64
          - os: windows-latest
            goos: windows
            goarch: amd64
            ext: ".exe"

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Extract version
        id: version
        run: echo "VERSION=$(git describe --tags --always --dirty)" >> $GITHUB_ENV

      - name: Build
        run: |
          OUTFILE=abb_tts-${{ matrix.goos }}-${{ env.VERSION }}-${{ matrix.goarch }}${{ matrix.ext }}
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
            go build -ldflags="-s -w -X main.version=${{ env.VERSION }}" \
            -o $OUTFILE .

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ matrix.goos }}-${{ matrix.goarch }}
          path: abb_tts-*
```

#### Release Workflow (`.github/workflows/release.yaml`)

```yaml
name: Release abb_tts

on:
  release:
    types: [published]

permissions:
  contents: write

jobs:
  test:
    name: Run Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: sudo apt-get install -y ffmpeg espeak
      - run: go test -v -race ./...

  build:
    name: Build
    needs: test
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
          - os: macos-latest
            goos: darwin
            goarch: amd64
          - os: macos-latest
            goos: darwin
            goarch: arm64
          - os: windows-latest
            goos: windows
            goarch: amd64
            ext: ".exe"

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Get version
        run: |
          VERSION="${GITHUB_REF_NAME#v}"
          echo "VERSION=$VERSION" >> $GITHUB_ENV

      - name: Build
        run: |
          OUTFILE=abb_tts-${{ matrix.goos }}-${{ env.VERSION }}-${{ matrix.goarch }}${{ matrix.ext }}
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
            go build -ldflags="-s -w -X main.version=${{ env.VERSION }}" \
            -o $OUTFILE .

      - uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ matrix.goos }}-${{ matrix.goarch }}
          path: abb_tts-*

  upload-release:
    name: Upload to Release
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/download-artifact@v4

      - name: Upload assets
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          for file in binaries-*/*; do
            gh release upload ${{ github.event.release.tag_name }} "$file"
          done
```

### Makefile

```makefile
# Makefile for abb_tts

BINARY_NAME=abb_tts
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"

.PHONY: all build clean test deps run

all: deps build

build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	@echo "Running tests..."
	go test -v -race ./...

deps:
	go mod tidy
	go mod download

run: build
	./$(BINARY_NAME) --no-browser

dev:
	go run . --no-browser --restart --log-level DEBUG

# Code quality
fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet
	@echo "Lint complete"
```

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes following existing code style
4. Test thoroughly
5. Submit a pull request

---

## License

[To be determined]

---

## Changelog

### v0.1.1 (2026-01-20)

#### Bug Fix: Line Breaks Lost in Output Text Files

**Problem**: Text files produced by the parser, sanitizer, and normalizer were losing all line breaks. Original ebook paragraphs (e.g., `<p>` tags in FB2 files) were being merged into a single long line in the output `.txt` files.

**Root Cause**: The default pronunciation rule in `internal/sanitize/text.go` used the regex pattern `\s+` to normalize whitespace. This pattern matches ALL whitespace characters including newlines (`\n`), replacing them with a single space and destroying paragraph structure.

**Fix**: Changed the regex pattern from `\s+` to `[ \t]+` to only match horizontal whitespace (spaces and tabs), preserving newlines for paragraph breaks.

**Files Changed**:
- `internal/sanitize/text.go`: Fixed whitespace normalization regex

#### Enhancement: Single Newline Between Paragraphs

**Change**: Reduced paragraph spacing from double newline (`\n\n`) to single newline (`\n`) for tighter, more compact text output.

**Files Changed**:
- `internal/parser/fb2.go`: Changed `</p>`, `<title>`, `<subtitle>`, `<empty-line>` to use single newline; updated `reFB2Newlines` pattern from `\n{3,}` to `\n{2,}`
- `internal/parser/epub.go`: Updated `reNewlines` pattern from `\n{3,}` to `\n{2,}`
- `internal/parser/fb2_test.go`: Updated test expectations for single newline

---

### v0.1.0 (2026-01-12)

- Initial server-client architecture
- Web UI with job management
- WebSocket real-time updates
- Text chunking for TTS
- espeak local TTS support
- EPUB and FB2 parsing
