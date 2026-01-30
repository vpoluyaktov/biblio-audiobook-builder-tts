# Biblio Audiobook Builder TTS

> Part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) application suite

## Overview

**Biblio Audiobook Builder TTS** is a server-based application that converts eBooks (EPUB, FB2) into audiobooks using various Text-to-Speech engines. It provides a web interface for uploading books, monitoring conversion progress, and downloading completed audiobooks.

### Key Features

- **Server-Client Architecture**: Offloads CPU-intensive TTS processing to a remote server
- **Drop-and-Forget Model**: Start a conversion, close the browser, reconnect later to check progress
- **Multi-Client Support**: Multiple users can upload, monitor, and download jobs simultaneously
- **Multiple TTS Engines**: Local (eSpeak), cloud (Google, OpenAI), and self-hosted (Silero, OpenVoice, OpenTTS, RHVoice)
- **Real-time Progress**: WebSocket-based live updates on conversion status
- **M4B Output**: Generates audiobooks with chapter markers, metadata, and cover art
- **OPDS Integration**: Browse and convert books directly from Biblio Catalog via OPDS
- **Audiobookshelf Integration**: Auto-upload completed audiobooks

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web Browser (Client)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ Upload Form │  │  Jobs List  │  │  WebSocket Connection   │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │ HTTP / WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      ABB-TTS Server                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────┐  │
│  │ REST API │  │ WebSocket│  │  Worker  │  │  TTS Service   │  │
│  │          │  │   Hub    │  │  Queue   │  │  (Providers)   │  │
│  └──────────┘  └──────────┘  └──────────┘  └────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
   │   Silero    │     │   Google    │     │   eSpeak    │
   │  TTS Server │     │  Cloud TTS  │     │   (Local)   │
   └─────────────┘     └─────────────┘     └─────────────┘
```

## Project Structure

```
biblio-audiobook-builder-tts/
├── main.go                          # Application entry point
├── Specification.md                 # This file
├── internal/
│   ├── config/config.go             # Configuration management
│   ├── server/
│   │   ├── server.go                # HTTP server, REST API
│   │   ├── job.go                   # Job model
│   │   ├── worker.go                # Background job processor
│   │   ├── websocket.go             # WebSocket hub
│   │   ├── assets/                  # CSS, JavaScript
│   │   └── templates/               # HTML templates
│   ├── storage/db.go                # SQLite database layer
│   ├── tts/
│   │   ├── service.go               # TTS service interface
│   │   ├── adapter.go               # Chunking adapter
│   │   ├── chunker.go               # Text chunking
│   │   ├── silero_provider.go       # Silero TTS
│   │   ├── google_provider.go       # Google Cloud TTS
│   │   ├── openai_provider.go       # OpenAI TTS
│   │   └── espeak_provider.go       # eSpeak (local)
│   ├── parser/
│   │   ├── epub_parser.go           # EPUB format
│   │   └── fb2_parser.go            # FB2 format
│   ├── audio/                       # M4B builder, FFmpeg wrappers
│   ├── normalize/                   # Number-to-words conversion
│   └── ssml/                        # SSML text processing
└── output/                          # Generated audiobooks
```

## REST API

### Core Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/` | Web interface |
| `POST` | `/api/upload` | Upload book and create job |
| `GET` | `/api/jobs` | List all jobs |
| `GET` | `/api/jobs/{id}` | Get job details |
| `DELETE` | `/api/jobs/{id}` | Delete/cancel job |
| `GET` | `/api/jobs/{id}/download` | Download completed audiobook |
| `WS` | `/api/ws` | WebSocket for real-time updates |

### Provider Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/providers` | List TTS providers |
| `GET` | `/api/providers/{id}` | Get provider details |
| `PUT` | `/api/providers/{id}` | Update provider config |
| `POST` | `/api/providers/{id}/test` | Test provider connection |
| `GET` | `/api/voices` | List available voices |

### OPDS Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/opds/sources` | List OPDS sources |
| `POST` | `/api/opds/sources` | Add OPDS source |
| `GET` | `/api/opds/browse?url=` | Browse OPDS catalog |
| `POST` | `/api/opds/download` | Download from OPDS |

## TTS Providers

| Provider | Type | Max Chunk | SSML | Notes |
|----------|------|-----------|------|-------|
| **eSpeak** | Local | 5000 | No | Free, low quality |
| **Silero** | Self-hosted | 900 | Yes | High quality, multiple languages |
| **OpenTTS** | Self-hosted | 2000 | No | Multiple engines |
| **RHVoice** | Self-hosted | 2000 | Yes | Russian, Ukrainian, English |
| **Google Cloud** | Cloud | 4000 | Yes | WaveNet, Neural2 voices |
| **OpenAI** | Cloud | 4000 | No | High quality |
| **Azure** | Cloud | 4000 | Yes | Neural voices |

### Provider Configuration

Each provider is stored in the `providers` table with:
- **Connection**: URL (self-hosted) or API key (cloud)
- **Performance**: `tts_workers`, `max_chunk_size`
- **Processing**: `normalize_numbers`, `ssml_support`

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ABB_TTS_HOST` | Server host | `0.0.0.0` |
| `ABB_TTS_PORT` | Server port | `80` |
| `ABB_TTS_BASE_PATH` | URL base path for path-based routing | `/abb-tts` |
| `ABB_TTS_SERVER_URL` | Silero TTS URL | `http://tts-silero:80/tts-silero` |
| `ABB_TTS_OPDS_SERVER_URL` | Biblio Catalog URL | `http://biblio-catalog:80/catalog` |

### Command Line Flags

```bash
./biblio-audiobook-builder-tts [flags]

Flags:
  --config string      Config file path
  --port string        Server port
  --host string        Server host
  --no-browser         Don't open browser
  --restart            Kill existing process
  --log-level string   DEBUG, INFO, WARN, ERROR
```

## Development Status

### Completed ✅

- HTTP server with REST API and WebSocket
- Job management (create, list, delete, progress tracking)
- Book parsing (EPUB, FB2)
- TTS providers: eSpeak, Silero, OpenTTS, RHVoice, Google, OpenAI
- Text chunking with per-provider max chunk size
- Number normalization (English, Russian)
- SSML processing with sentence/paragraph pauses
- M4B audiobook generation with chapters
- Cover image extraction and embedding
- OPDS catalog browsing
- Audiobookshelf integration
- Settings UI with provider management
- SQLite persistence
- Silent audio generation for chapters with no speakable content (maintains chapter alignment)
- Chapter gap silence insertion (configurable via `chapter_gap_seconds` setting)
- Roman numeral normalization for chapter/part titles (I, II, III, IV, etc.)

### In Progress 🔄

- Parallel chapter processing
- Test voice UI improvements
- Part separator silence detection (see below)

---

## Feature: Part Separator Silence Detection

### Problem Statement

Some poorly formatted eBooks have only one or two chapters, but the actual content is divided into multiple parts/scenes separated by visual markers such as:
- `* * *` or `***`
- `---` or `- - -`
- `• • •`
- `~ ~ ~`
- Multiple blank lines
- Other decorative separators

Currently, the `ChapterGapSeconds` setting only inserts silence between chapters. For books with minimal chapter structure, this means no pauses are inserted between logical parts, resulting in a continuous audio stream that lacks natural breaks.

### Goal

Detect part separators within chapter content and insert the same `ChapterGapSeconds` silence (or a configurable `PartGapSeconds`) between parts, improving the listening experience for poorly structured eBooks.

### Solution Options

#### Option 1: Regex-Based Separator Detection in Parser

**Description**: Modify the EPUB/FB2 parsers to detect common separator patterns and split chapter content into sub-parts.

**Implementation**:
- Add regex patterns to detect separators: `^\s*[\*\-•~]{3,}\s*$`, `^\s*\*\s+\*\s+\*\s*$`, etc.
- When a separator is detected, split the chapter `Content` into multiple segments
- Add a new field to `Chapter` struct: `Parts []string` or use a separator marker

**Pros**:
- Clean separation at parse time
- Separator patterns are removed from text (won't be spoken)
- Easy to extend with new patterns

**Cons**:
- Requires modifying parser output structure
- May need to handle edge cases (separators in dialogue, code blocks, etc.)

#### Option 2: SSML Break Injection During Text Processing

**Description**: Detect separators during SSML processing and inject `<break time="Xs"/>` tags.

**Implementation**:
- In `internal/ssml/` processing, scan for separator patterns
- Replace separator lines with SSML break tags
- Leverage existing SSML infrastructure

**Pros**:
- No parser changes needed
- Works with existing SSML-capable TTS providers
- Natural integration with pause handling

**Cons**:
- Only works with SSML-capable providers (Silero, Google, Azure, RHVoice)
- eSpeak and OpenAI don't support SSML breaks

#### Option 3: Audio-Level Silence Insertion in Worker

**Description**: Detect separators in the worker during TTS processing and insert silence audio segments.

**Implementation**:
- Before sending text to TTS, scan for separator patterns
- Split text at separators, generate TTS for each segment
- Insert silence audio (using existing `generateSilence()`) between segments
- Concatenate audio segments

**Pros**:
- Works with all TTS providers (no SSML dependency)
- Uses existing silence generation infrastructure
- Consistent behavior across providers

**Cons**:
- More complex audio handling
- Increases number of TTS API calls (one per segment)

#### Option 4: Hybrid Approach (Recommended)

**Description**: Combine parser-level detection with audio-level silence insertion.

**Implementation**:
1. Add a `PartSeparatorPatterns` config option (list of regex patterns)
2. Add `PartGapSeconds` config option (defaults to `ChapterGapSeconds`)
3. In parser: detect and mark separators but keep content in single chapter
4. In worker: when processing chapter text, split at separator markers and insert silence

**Pros**:
- Works with all TTS providers
- Configurable patterns for different book styles
- Separators are removed from spoken text
- Minimal parser changes (just marking, not restructuring)

**Cons**:
- Slightly more complex than single-layer solutions

### Recommended Approach

**Option 4 (Hybrid)** is recommended because:
1. It works universally with all TTS providers
2. It's configurable for different separator styles
3. It cleanly removes separators from spoken output
4. It reuses existing silence generation code

### Configuration

New settings to add:
```go
// Config additions
PartGapSeconds         int      `mapstructure:"part_gap_seconds"`          // Silence between parts (default: same as chapter_gap_seconds)
DetectPartSeparators   bool     `mapstructure:"detect_part_separators"`    // Enable part separator detection
PartSeparatorPatterns  []string `mapstructure:"part_separator_patterns"`   // Custom regex patterns
```

Default patterns:
```go
var DefaultPartSeparatorPatterns = []string{
    `^\s*\*\s*\*\s*\*\s*$`,           // * * *
    `^\s*\*{3,}\s*$`,                  // ***
    `^\s*-\s*-\s*-\s*$`,               // - - -
    `^\s*-{3,}\s*$`,                   // ---
    `^\s*•\s*•\s*•\s*$`,               // • • •
    `^\s*~\s*~\s*~\s*$`,               // ~ ~ ~
    `^\s*#\s*#\s*#\s*$`,               // # # #
    `^\s*\.\s*\.\s*\.\s*$`,            // . . .
}
```

---

### Future Enhancements

- Azure TTS integration
- Audio normalization
- Noise reduction
- Additional language support for number normalization

## Dependencies

### System Requirements

- **ffmpeg**: Audio processing, M4B creation
- **ffprobe**: Audio file inspection
- **espeak** (optional): Local TTS

### Go Modules

- `github.com/google/uuid` - Job IDs
- `github.com/gorilla/websocket` - WebSocket support
- `github.com/mattn/go-sqlite3` - SQLite database

## Docker Deployment

Part of BiblioHub Docker Swarm stack. Access via `http://localhost:9900/abb-tts/`

```yaml
abb-tts:
  image: vpoluyaktov/bibliohub-audiobook-builder-tts:dev-latest
  environment:
    - ABB_TTS_HOST=0.0.0.0
    - ABB_TTS_PORT=80
    - ABB_TTS_BASE_PATH=/abb-tts
    - ABB_TTS_SERVER_URL=http://tts-silero:80/tts-silero
    - ABB_TTS_OPDS_SERVER_URL=http://biblio-catalog:80/catalog
    - ABB_TTS_TEMP_DIR=/data
    - ABB_TTS_LOG_FILE=/logs/abb_tts.log
  volumes:
    - ./data/abb_tts/db:/db
    - ./data/abb_tts/temp:/data
    - ./data/abb_tts/logs:/logs
```

### Docker Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ABB_TTS_HOST` | Server host | `0.0.0.0` |
| `ABB_TTS_PORT` | Server port | `80` |
| `ABB_TTS_BASE_PATH` | URL base path for path-based routing | `/abb-tts` |
| `ABB_TTS_TEMP_DIR` | Working directory for ebook downloads, chapter files, and audiobooks | `/data` |
| `ABB_TTS_LOG_FILE` | Log file path | `/logs/abb_tts.log` |
| `ABB_TTS_SERVER_URL` | Silero TTS server URL | `http://tts-silero:80/tts-silero` |
| `ABB_TTS_OPDS_SERVER_URL` | Biblio Catalog URL | `http://biblio-catalog:80/catalog` |

---

*Last updated: 2026-01-30*
