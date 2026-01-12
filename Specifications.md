# Audiobook Builder TTS - Specifications

## Overview

**Audiobook Builder TTS (abb_tts)** is a server-based application that converts eBooks (EPUB, FB2) into audiobooks using various Text-to-Speech engines. It provides a web interface for uploading books, monitoring conversion progress, and downloading completed audiobooks.

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
abb_tts/
├── main.go                     # Application entry point
├── abb_tts.config.yaml         # Configuration file
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
./abb_tts [flags]

Flags:
  --config string      Path to configuration file (default "abb_tts.config.yaml")
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

### In Progress 🔄

| Feature | Status | Notes |
|---------|--------|-------|
| - | - | No features currently in progress |

### Not Started ❌

| Feature | Priority | Notes |
|---------|----------|-------|
| Audio Normalization | Low | Consistent volume levels |
| Noise Reduction | Low | Clean up TTS artifacts |
| Cloud TTS (Google) | Low | Needs API integration |
| Cloud TTS (Azure) | Low | Needs API integration |
| Job Persistence | Low | Database storage |

---

## Future Enhancements

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

### v0.1.0 (2026-01-12)

- Initial server-client architecture
- Web UI with job management
- WebSocket real-time updates
- Text chunking for TTS
- espeak local TTS support
- EPUB and FB2 parsing
