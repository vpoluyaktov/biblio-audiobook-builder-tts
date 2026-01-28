# Biblio Audiobook Builder TTS

> Part of the [BiblioHub](https://github.com/vpoluyaktov/BiblioHub) application suite

## Overview

**Biblio Audiobook Builder TTS** is a server-based application that converts eBooks (EPUB, FB2) into audiobooks using various Text-to-Speech engines. It provides a web interface for uploading books, monitoring conversion progress, and downloading completed audiobooks.

### Key Features

- **Server-Client Architecture**: Offloads CPU-intensive TTS processing to a remote server
- **Drop-and-Forget Model**: Start a conversion, close the browser, reconnect later to check progress
- **Multi-Client Support**: Multiple users can upload, monitor, and download jobs simultaneously
- **Multiple TTS Engines**: Local (eSpeak), cloud (Google, OpenAI, Azure), and self-hosted (Silero, OpenTTS, RHVoice)
- **Real-time Progress**: WebSocket-based live updates on conversion status
- **M4B Output**: Generates audiobooks with chapter markers, metadata, and cover art
- **OPDS Integration**: Browse and convert books directly from OPDS catalogs
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
| `ABB_TTS_PORT` | Server port | `9901` |
| `ABB_TTS_SERVER_URL` | Silero TTS URL | `http://tts-silero:9902` |
| `ABB_TTS_OPDS_SERVER_URL` | OPDS server URL | `http://opds-server:9903` |

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

### In Progress 🔄

- Parallel chapter processing
- Test voice UI improvements

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

Part of BiblioHub Docker Swarm stack:

```yaml
abb-tts:
  image: vpoluyaktov/bibliohub-audiobook-builder-tts:dev-latest
  ports:
    - "9901:9901"
  environment:
    - ABB_TTS_SERVER_URL=http://tts-silero:9902
    - ABB_TTS_OPDS_SERVER_URL=http://opds-server:9903
    - ABB_TTS_TEMP_DIR=/data
  volumes:
    - ./data/abb_tts/db:/db
    - ./data/abb_tts/data:/data
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ABB_TTS_HOST` | Server host | `0.0.0.0` |
| `ABB_TTS_PORT` | Server port | `9901` |
| `ABB_TTS_TEMP_DIR` | Working directory for ebook downloads, chapter files, and audiobooks | `./temp` |
| `ABB_TTS_LOG_FILE` | Log file path | `biblio-audiobook-builder-tts.log` |
| `ABB_TTS_SERVER_URL` | Silero TTS server URL | `http://tts-silero:9902` |
| `ABB_TTS_OPDS_SERVER_URL` | OPDS server URL | `http://opds-server:9903` |

---

## Recent Changes

### fix/auto-detect-sample-rate (2026-01-25)

**Problem**: M4B audiobooks were being encoded with incorrect pitch when the source WAV sample rate didn't match the configured output sample rate. For example, Silero TTS outputs at 48000 Hz, but the M4B builder was configured to use 44100 Hz, causing pitch to drop.

**Solution**: 
- Removed configurable `bit_rate_kbs` and `sample_rate_hz` settings from the application
- M4B builder now auto-detects the sample rate from the first source WAV file
- FFmpeg determines optimal bitrate automatically
- Removed Audio Settings section from the Settings UI

**Files Changed**:
- `internal/audio/m4b.go` - Added `getAudioSampleRate()` function, auto-detect sample rate in `BuildFromFiles()`
- `internal/storage/db.go` - Removed `BitRateKbs` and `SampleRateHz` from Config struct
- `internal/config/config.go` - Removed bit rate and sample rate config fields
- `internal/server/settings.go` - Removed bit rate and sample rate from settings API
- `internal/server/worker.go` - Updated M4B options to not pass sample rate
- `internal/server/templates/index.html` - Removed Audio Settings section from UI
- `internal/server/assets/app.js` - Removed bit rate and sample rate from settings JS

---

### fix/silent-wav-sample-rate (2026-01-25)

**Problem**: Silent WAV files generated for empty chapters were hardcoded to 44100 Hz. When the first chapter of a book was empty (e.g., "Chapter 1" with no speakable content), the M4B auto-detection would pick up this 44100 Hz file instead of the TTS provider's actual sample rate (e.g., 48000 Hz for Silero), causing pitch issues in the final M4B audiobook.

**Solution**:
- Added `sample_rate` field to the `providers` table in the database schema
- Set provider-specific sample rates: Silero (48000 Hz), OpenVoice/Google (44100 Hz), Azure/OpenAI/RHVoice (24000 Hz), eSpeak/OpenTTS (22050 Hz)
- Updated silent WAV generation to use the provider's sample rate instead of hardcoded 44100 Hz
- Added database migration to populate sample rates for existing providers

**Files Changed**:
- `internal/storage/db.go` - Added `sample_rate` column to providers table, added `SampleRate` field to `TTSProvider` struct, updated all SQL queries, added migration
- `internal/server/worker.go` - Updated silent WAV generation to fetch and use provider's sample rate

**Impact**: Ensures consistent sample rates throughout the audiobook conversion pipeline, preventing pitch distortion when books contain empty chapters.

---

### fix/path-based-routing - Handler Path Parsing (2026-01-28)

**Problem**: When using path-based routing with a base path (e.g., `/abb-tts`), several API handlers were failing with errors like "Unknown action" (400) or "Provider not found" (404). This specifically affected the Silero TTS provider configuration in the settings window, where testing the connection and saving provider settings would fail.

**Root Cause**: Multiple request handlers were using incorrect path parsing logic that assumed the URL path started with the API prefix (e.g., `/api/providers/`), but when a base path was configured, the actual path was `/abb-tts/api/providers/...`. The handlers were slicing the path at the wrong position, causing incorrect extraction of IDs and actions.

**Solution**:
- Updated `handleJob()` to use `strings.Index()` to find `/api/jobs/` prefix position
- Updated `handleProviderByID()` to use `strings.Index()` to find `/api/providers/` prefix position
- Updated `handlePreviewByID()` to use `strings.Index()` to find `/api/preview/` prefix position
- Updated `handleNoun()` to use `strings.Index()` to find `/api/nouns/` prefix position
- All handlers now correctly extract remaining path segments after finding the prefix, regardless of base path

**Example Fix**:
```go
// Before (incorrect):
remaining := path[len(prefix):]  // Assumes path starts with prefix

// After (correct):
idx := strings.Index(path, prefix)
if idx == -1 || len(path) <= idx+len(prefix) {
    http.Error(w, "Invalid path", http.StatusBadRequest)
    return
}
remaining := path[idx+len(prefix):]  // Finds prefix position first
```

**Files Changed**:
- `internal/server/server.go` - Fixed `handleJob()`, `handleProviderByID()`, `handlePreviewByID()`
- `internal/server/noun_handlers.go` - Fixed `handleNoun()`

**Impact**: Resolves all path-based routing issues for provider configuration, job management, preview operations, and noun dictionary management when using a base path.

---

### fix/opds-cover-image-url - OPDS Cover Image Display (2026-01-28)

**Problem**: When downloading books from OPDS catalogs, the cover image was not displayed in the book preview dialog. The preview was returning a cover URL like `/api/preview/{id}/cover` which resulted in a 404 error, while the OPDS catalog's cover image was available and working through the proxy endpoint at `/api/opds/proxy?url=...`.

**Root Cause**: The `handleOPDSDownload` endpoint was creating a preview from the downloaded book file, which attempted to extract the cover image from the book content. However, for OPDS books, the cover image URL from the OPDS catalog entry was not being passed through or utilized. The preview was setting `CoverImageURL` to `/api/preview/{id}/cover`, but this endpoint only works when the cover image is extracted from the book file and stored in the preview store. For OPDS books, the cover should use the catalog's cover URL through the proxy.

**Solution**:
- Modified the JavaScript `downloadAndConvert()` function to include the `cover_url` from the OPDS catalog entry in the download request
- Updated the `handleOPDSDownload` endpoint to accept the `cover_url` parameter
- When a cover URL is provided from OPDS, override the preview's `CoverImageURL` to use the proxy endpoint with proper URL encoding
- Added `basePath` parameter to `PreviewStore` to ensure cover URLs for uploaded files also include the base path
- Created `apiURL()` helper method in the `Server` struct to centralize base path handling in the backend
- Frontend already had `apiUrl()` helper function that correctly handles base path for all API calls

**Files Changed**:
- `internal/server/assets/app.js` - Added `cover_url` field to the OPDS download request payload
- `internal/server/opds_handlers.go` - Added `CoverURL` field to request struct, uses `apiURL()` helper for cover URL construction
- `internal/server/preview.go` - Added `basePath` field to `PreviewStore`, updated `NewPreviewStore()` signature, fixed cover URL generation
- `internal/server/server.go` - Added `apiURL()` helper method, updated `NewPreviewStore()` call to pass `basePath`
- `internal/server/preview_test.go` - Updated all test calls to pass empty `basePath` parameter

**Impact**: 
- OPDS book previews now correctly display cover images from the OPDS catalog
- Regular uploaded files also have correct cover URLs with base path
- Centralized base path handling reduces code duplication and makes future maintenance easier
- Both frontend and backend now have consistent helper methods for URL construction

---

### fix/sanitize-ellipsis - Ellipsis in Filenames (2026-01-28)

**Problem**: Book titles starting with `...` (ellipsis) created problematic file paths that some audiobook servers (like Audiobookshelf) couldn't handle properly. Example: `...И двадцать четыре жемчужины` would create paths like `/data/И_двадцать_четыре_жемчужины/Людмила_Васильева_-_...И_двадцать_четыре_жемчужины, Part 1.m4b`.

**Solution**: Added `...` (triple dot ellipsis) and `..` (double dot) to the list of sanitized substrings in the filename sanitizer. These are replaced with underscores, which are then collapsed with other underscores and trimmed from the ends.

**Files Changed**:
- `internal/sanitize/filename.go` - Added `...` and `..` to the `strings.NewReplacer` list

**Impact**: Audiobook files with ellipsis in their titles will now have clean, compatible filenames.

---

*Last updated: 2026-01-28*
