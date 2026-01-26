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

*Last updated: 2026-01-25*
