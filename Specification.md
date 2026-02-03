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
- Part separator silence detection (detects `***`, `---`, `• • •`, etc. and inserts silence between parts)
- SSML pause patterns for dashes and ellipsis (converts ` - ` and `...` to `<break>` tags for TTS)
- SSML break tag support for sentence and paragraph pauses (configurable durations instead of `<p>`/`<s>` tags)
- SSML-first chunking to preserve pauses across chunk boundaries
- SSML-aware chunker that never splits SSML tags
- Comprehensive sentence separator support (!!!, ???, ?!, ellipsis, Unicode punctuation)
- SSML debug files for troubleshooting (`.ssml.txt` files show text with breaks before chunking)

### In Progress 🔄

- Parallel chapter processing
- Test voice UI improvements
- User authentication (internal + biblio-auth modes)

---

## Feature: Title Break SSML Support 🔄 IN PROGRESS

### Problem Statement

In FB2 books, `<title>` tags mark chapter titles and section headings within the content. Currently, these titles are converted to plain text with simple newlines, resulting in no extra pause after the title is spoken. This makes the transition from title to content feel rushed and unnatural.

Example FB2 structure:
```xml
<section>
  <title>
    <p>Глава 1</p>
  </title>
  <p>Мертвяк неловко перевалился через высокий забор...</p>
</section>
```

Currently outputs:
```
Глава 1
Мертвяк неловко перевалился через высокий забор...
```

The TTS reads "Глава 1" and immediately continues with the content, with only a standard paragraph pause.

### Goal

Add an extra SSML pause after `<title>` tags to create a more natural listening experience, similar to how a human narrator would pause after announcing a chapter or section title.

### Solution: Hybrid Marker Approach

**Architecture**:
1. **FB2 Parser**: Insert `{{TITLE_BREAK}}` marker after `</title>` tags
2. **SSML Processor**: Convert marker to `<break time="Xms"/>` tag for SSML-capable providers
3. **Configuration**: Add `TitleBreakMs` setting (default: 500ms)

**Implementation**:

1. **Add TitleBreakMarker constant** (`internal/normalize/separator.go`):
   ```go
   const TitleBreakMarker = "\n{{TITLE_BREAK}}\n"
   ```

2. **Modify FB2 parser** (`internal/parser/fb2.go`):
   ```go
   text = reFB2TitleClose.ReplaceAllString(text, TitleBreakMarker)
   ```

3. **Add config option** (`internal/config/config.go`):
   ```go
   TitleBreakMs int `mapstructure:"title_break_ms"` // Default: 500ms
   ```

4. **Update SSML processor** (`internal/ssml/ssml.go`):
   - Add `TitleBreakMs` to `SSMLOptions`
   - Convert `{{TITLE_BREAK}}` marker to `<break time="Xms"/>` tag

**Output with feature**:
```
Глава 1 <break time="500ms"/>
Мертвяк неловко перевалился через высокий забор...
```

### Benefits

- ✅ Natural pause after chapter/section titles
- ✅ Configurable pause duration
- ✅ Works with all SSML-capable TTS providers (Silero, Google, Azure, RHVoice)
- ✅ Consistent with existing marker pattern (like `{{PART_SEPARATOR}}`)
- ✅ Extensible to EPUB parser in the future

### Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `title_break_ms` | 500 | Pause duration after titles in milliseconds |

### Files Modified

- `internal/normalize/separator.go` - Add `TitleBreakMarker` constant
- `internal/parser/fb2.go` - Insert marker after `</title>` tags
- `internal/config/config.go` - Add `TitleBreakMs` config option
- `internal/ssml/ssml.go` - Convert marker to SSML break tag
- `internal/tts/service.go` - Pass `TitleBreakMs` in conversion options

**Date:** 2026-02-02

---

## Configuration Update: TTS Break Duration Defaults ✅ UPDATED

### Change Summary

Updated default SSML break durations to provide faster, more natural-sounding speech pacing:

**Previous defaults:**
- Sentence break: 500ms
- Paragraph break: 800ms
- Dash/ellipsis break: 300ms

**New defaults:**
- Sentence break: 300ms
- Paragraph break: 350ms
- Dash/ellipsis break: 250ms

### Rationale

The original defaults were based on general speech research but resulted in overly long pauses that slowed down audiobook playback. The new defaults provide:
- Faster pacing while maintaining natural speech rhythm
- Better alignment with professional audiobook narration speeds
- Reduced total audiobook duration without sacrificing clarity

### Impact

- New installations will use the updated defaults automatically
- Existing deployments retain their current settings (stored in database)
- Users can still customize all pause durations via the web UI

**Files modified:**
- `internal/config/config.go` - Updated default values and comments

**Date:** 2026-02-02

---

## Feature: SSML Break Tag Refactoring ✅ IMPLEMENTED

### Problem Statement

The current implementation uses SSML `<p>` (paragraph) and `<s>` (sentence) tags to add natural pauses between paragraphs and sentences. However, some TTS providers don't support these tags properly, while they do support `<break time="Xms"/>` tags correctly.

### Goal

Replace `<p>` and `<s>` SSML tags with explicit `<break>` tags with configurable durations based on average human speech patterns.

### Implementation Plan

**Recommended break durations** (based on speech research):
- **Sentence break**: 500ms (typical pause between sentences)
- **Paragraph break**: 800ms (longer pause for paragraph transitions)
- **Inline dash/ellipsis**: 300ms (already implemented)

**Implementation**:

**Status**: ✅ Implemented

**Files modified**:
- `internal/ssml/ssml.go` - Refactored SSML generation to use `<break>` tags instead of `<p>`/`<s>` wrapper tags
- `internal/config/config.go` - Added `SentenceBreakMs` and `ParagraphBreakMs` config fields
- `internal/storage/db.go` - Added storage support for new config fields
- `internal/tts/service.go` - Updated `ConversionOptions` struct
- `internal/tts/adapter.go` - Updated to pass new break durations to SSML generation
- `internal/server/worker.go` - Updated TTS conversion calls with new options
- `internal/server/settings.go` - Moved pause settings to TTS section, added new fields
- `internal/server/templates/index.html` - Moved pause settings to TTS Settings tab, added new UI controls
- `internal/server/assets/app.js` - Updated form population and collection for new fields

**Configuration**:
```go
SentenceBreakMs   int  `mapstructure:"sentence_break_ms"`   // Default: 500ms
ParagraphBreakMs  int  `mapstructure:"paragraph_break_ms"`  // Default: 800ms
```

**How it works**:
1. Instead of wrapping text in `<p>` and `<s>` tags, the SSML generator now inserts `<break time="Xms"/>` tags
2. Sentence breaks are inserted between sentences within a paragraph
3. Paragraph breaks are inserted between paragraphs
4. Setting a value to 0 disables that type of break
5. All pause settings (sentence, paragraph, dash, ellipsis) are now consolidated in the TTS Settings tab
6. Maintains backward compatibility with legacy `UseSentencePauses` flag

---

## Feature: SSML-First Chunking ✅ IMPLEMENTED

### Problem Statement

The previous implementation had a critical flaw where SSML break tags were added to each chunk independently. This caused pauses to be lost at chunk boundaries:

**Example of the bug:**
```
Original text: "Sentence 1. Sentence 2. Sentence 3."
Chunked as:
  Chunk 1: "Sentence 1. Sentence 2."
  Chunk 2: "Sentence 3."

SSML wrapping (per chunk):
  Chunk 1: <speak>Sentence 1.<break time="500ms"/>Sentence 2.</speak>
  Chunk 2: <speak>Sentence 3.</speak>

Result: Pause between Sentence 1 and 2 ✓
        NO pause between Sentence 2 and 3 ❌ (chunk boundary)
```

### Solution

Implement **SSML-first chunking**: Add SSML break tags to the entire chapter text BEFORE chunking, then chunk the SSML-enriched text.

**New flow:**
```
1. Get chapter text from parser
2. Add ALL SSML break tags to the full text (if SSML enabled)
3. Chunk the SSML-enriched text (chunker is SSML-aware, won't break tags)
4. Wrap each chunk in <speak></speak> tags only
5. Send to TTS
```

### Implementation

**Status**: ✅ Implemented

**Key changes:**

1. **New function `AddSSMLBreaks()`** (`internal/ssml/ssml.go`)
   - Adds break tags to text without `<speak>` wrapper
   - Used for pre-chunking SSML processing

2. **SSML-aware chunker** (`internal/tts/chunker.go`)
   - Added `isInsideSSMLTag()` helper function
   - Modified `findBreakPoint()` to never break inside SSML tags
   - Ensures tags like `<break time="500ms"/>` remain intact

3. **Updated adapter** (`internal/tts/adapter.go`)
   - Calls `AddSSMLBreaks()` on entire text before chunking
   - Chunks now contain SSML breaks embedded in the text
   - Only wraps chunks in `<speak>` tags (breaks already present)

**Benefits:**
- ✅ All sentence/paragraph pauses preserved across chunk boundaries
- ✅ No pauses lost at chunk boundaries
- ✅ Simpler implementation (no chunk metadata needed)
- ✅ SSML tags never split across chunks
- ✅ Works within TTS engine chunk size limits (900 chars)

**Example with fix:**
```
Original text: "Sentence 1. Sentence 2. Sentence 3."

After AddSSMLBreaks():
"Sentence 1.<break time="500ms"/>Sentence 2.<break time="500ms"/>Sentence 3."

After chunking (if needed):
  Chunk 1: "Sentence 1.<break time="500ms"/>Sentence 2."
  Chunk 2: "<break time="500ms"/>Sentence 3."

After wrapping:
  Chunk 1: <speak>Sentence 1.<break time="500ms"/>Sentence 2.</speak>
  Chunk 2: <speak><break time="500ms"/>Sentence 3.</speak>

Result: ALL pauses preserved ✅
```

---

## Feature: Part Separator Silence Detection ✅ IMPLEMENTED

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

### Implementation (Option 4 - Hybrid Approach)

**Status**: ✅ Implemented

**Files modified/created**:
- `internal/config/config.go` - Added `PartGapSeconds` and `DetectPartSeparators` config fields
- `internal/normalize/separator.go` - Part separator detection module
- `internal/normalize/separator_test.go` - Unit tests for separator detection
- `internal/audio/m4b.go` - Added `ConcatWAVFiles()` function
- `internal/server/worker.go` - Integrated separator detection and silence insertion

**Configuration**:
```go
// Config fields
PartGapSeconds         int     `mapstructure:"part_gap_seconds"`          // Silence between parts (default: 2 seconds)
DetectPartSeparators   bool    `mapstructure:"detect_part_separators"`    // Enable part separator detection (default: true)
```

**Default patterns detected**:
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
    // ... and more (see separator.go)
}
```

**How it works**:
1. When `DetectPartSeparators` is enabled (default: true), the worker scans chapter content for separator patterns
2. Separators are replaced with internal markers
3. Content is split at markers into parts
4. Each part is converted to speech separately
5. Silence audio (duration: `PartGapSeconds`) is inserted between parts
6. All parts are concatenated into the final chapter audio file

---

## Feature: SSML Pause Patterns ✅ IMPLEMENTED

### Problem Statement

Some TTS providers don't naturally pause on certain punctuation marks like inline dashes (`цель - дыра`) or ellipsis (`там... были`). This results in words running together without natural pauses.

### Solution

Convert pause-triggering punctuation to SSML `<break>` tags when the TTS provider supports SSML.

**Status**: ✅ Implemented

**Files modified/created**:
- `internal/ssml/ssml.go` - Added `DefaultPausePatternDefs` and pause conversion logic
- `internal/config/config.go` - Added `ConvertDashesToBreaks` and `DashBreakDurationMs` config fields
- `internal/storage/db.go` - Added storage support for new config fields
- `internal/server/settings.go` - Added UI settings support
- `internal/server/templates/index.html` - Added "Dash Pause (SSML)" settings section

**Configuration**:
```go
ConvertDashesToBreaks   bool    `mapstructure:"convert_dashes_to_breaks"`  // Enable pause conversion (default: true)
DashBreakDurationMs     int     `mapstructure:"dash_break_duration_ms"`    // Pause duration in ms (default: 300)
```

**Pause patterns**:
```go
var DefaultPausePatternDefs = []PausePatternDef{
    // Ellipsis: "..." or "…" - adds pause after ellipsis
    {`\.{3,}`, `...%s`},
    {`…`, `…%s`},
    // Inline dashes: " - ", " — ", " – " (with surrounding spaces)
    {`\s+[-—–]\s+`, ` %s `},
}
```

**How it works**:
1. When `ConvertDashesToBreaks` is enabled and the TTS provider supports SSML
2. Text is scanned for pause patterns (dashes, ellipsis)
3. Patterns are replaced with the original punctuation plus a `<break time="Xms"/>` tag
4. Example: `"цель - дыра"` → `"цель <break time="300ms"/> дыра"`

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

## Feature: Comprehensive Sentence Separator Support ✅ IMPLEMENTED

### Problem Statement

Ebooks often use varied punctuation styles for dramatic effect:
- Multiple exclamation marks: `!!!`, `!!`
- Multiple question marks: `???`, `??`
- Combined punctuation: `?!`, `!?`
- Ellipsis variations: `...`, `....`, `.....`
- Unicode punctuation: Armenian `։`, Arabic `؟`, Interrobang `‽`

The previous implementation only recognized single punctuation marks (`.`, `!`, `?`), causing these variations to be treated as mid-sentence punctuation, resulting in missing pauses.

### Solution

Updated the sentence separator regex pattern to recognize all common punctuation combinations found in ebooks.

### Implementation

**Status**: ✅ Implemented

**Changes made:**

1. **Updated sentence pattern** (`internal/ssml/ssml.go`):
   - Changed from `([.!?։؟])` to `([.!?։؟‽]+)`
   - The `+` quantifier matches one or more consecutive punctuation marks
   - Now treats ellipsis as sentence boundaries for natural pauses

2. **Supported separators:**
   - Single: `.` `!` `?`
   - Multiple: `!!` `!!!` `??` `???`
   - Combined: `?!` `!?`
   - Ellipsis: `...` `....` `.....`
   - Unicode: `։` `؟` `‽`

3. **Comprehensive test coverage:**
   - Created `ssml_sentence_test.go` with 20+ test cases
   - Tests all separator combinations
   - Verifies edge cases

**Example:**
```
Input: "What?! Really!!! Yes... Okay."
Output: What?!<break time="500ms"/>Really!!!<break time="500ms"/>Yes...<break time="500ms"/>Okay.
```

**Benefits:**
- ✅ Natural pauses after dramatic punctuation
- ✅ Better handling of literary styles
- ✅ Support for international ebooks
- ✅ Improved audiobook listening experience

---

## Feature: SSML Debug Files ✅ IMPLEMENTED

### Problem Statement

When troubleshooting SSML-first chunking, it's difficult to verify that SSML break tags are being added correctly to the text before chunking occurs. The only way to see the SSML was to inspect the actual TTS requests.

### Solution

Save debug files showing the text with SSML break tags before chunking, alongside the regular chapter text files.

### Implementation

**Status**: ✅ Implemented

**Changes made:**

1. **Debug file output** (`internal/server/worker.go`):
   - For each chapter, save two files:
     - `01_Chapter_Title.txt` - Raw chapter text (sanitized/normalized)
     - `01_Chapter_Title.ssml.txt` - Text with SSML break tags before chunking
   - Only created when SSML is enabled for the provider
   - Saved in the same temp directory as regular text files

2. **File location:**
   ```
   /home/ubuntu/git/biblio-hub/data/abb_tts/temp/[BookName]/
   ├── 01_Chapter_Title.txt
   ├── 01_Chapter_Title.ssml.txt
   ├── 02_Chapter_Title.txt
   ├── 02_Chapter_Title.ssml.txt
   └── ...
   ```

3. **Example SSML debug file content:**
   ```
   First sentence.<break time="500ms"/>Second sentence.<break time="500ms"/>
   
   <break time="800ms"/>New paragraph here.<break time="500ms"/>Another sentence.
   ```

**Benefits:**
- ✅ Easy verification of SSML-first chunking
- ✅ Visual confirmation that breaks are preserved
- ✅ Helpful for debugging pause issues
- ✅ No impact on production (debug files only)

---

## Feature: User Authentication 🔄 IN PROGRESS

### Problem Statement

ABB-TTS currently has no authentication, allowing anyone with network access to use the service. For standalone deployment and integration with BiblioHub stack, user authentication is required.

### Goal

Implement user authentication supporting two modes:
1. **Internal Mode** (`AUTH_MODE=internal`): Standalone deployment with local SQLite user database
2. **Biblio Auth Mode** (`AUTH_MODE=biblio-auth`): Integration with Biblio Auth for centralized authentication in BiblioHub stack

### Implementation Plan

**Phase 1: Core Authentication Infrastructure**
1. ✅ Create `internal/auth/` package with:
   - `biblioauth.go` - Biblio Auth client for JWT validation
   - `manager.go` - Auth manager supporting both modes
   - `middleware.go` - HTTP middleware for route protection
2. ✅ Add auth configuration to `internal/config/config.go`
3. ✅ Add user/session tables to database schema (`internal/storage/auth.go`)

**Phase 2: Server Integration**
4. ✅ Update `internal/server/server.go` to integrate auth middleware
5. ✅ Add auth API endpoints (login, logout, user info) - `internal/server/handlers_auth.go`
6. ✅ Protect all routes except health check and auth endpoints

**Phase 3: Frontend Integration**
7. ✅ Update frontend to handle authentication (`internal/server/assets/app.js`)
8. ✅ Add login UI for internal mode
9. ✅ Add redirect to Biblio Auth login for biblio-auth mode
10. ✅ Display current user info and logout button

**Configuration**:
```go
// Auth settings
AuthMode       string `mapstructure:"auth_mode"`        // "internal" or "biblio-auth"
BiblioAuthURL  string `mapstructure:"biblio_auth_url"`  // Biblio Auth service URL
```

**Environment Variables**:
| Variable | Description | Default |
|----------|-------------|---------|
| `ABB_TTS_AUTH_MODE` | Authentication mode (`internal` or `biblio-auth`) | `biblio-auth` |
| `ABB_TTS_BIBLIO_AUTH_URL` | Biblio Auth service URL | `http://biblio-auth:80/auth` |

**Authentication Flow (Biblio Auth Mode)**:
1. User accesses ABB-TTS → Check for `auth_token` cookie
2. No valid token → Redirect to `/auth/login?returnUrl=<current_url>`
3. User logs in at Biblio Auth → JWT token set as `auth_token` cookie
4. Redirect back to ABB-TTS → Validate token via `/auth/api/validate`
5. Token valid → Access granted

**Authentication Flow (Internal Mode)**:
1. User accesses ABB-TTS → Check for session cookie
2. No valid session → Show login form
3. User submits credentials → Validate against local database
4. Valid credentials → Create session, set cookie
5. Access granted

---

## Feature: Stress Marking Integration (Silero Stress) ⏳ PLANNED

### Problem Statement

Russian is a stress-timed language where word stress is not marked in standard orthography. Incorrect stress can:
- Change word meaning (e.g., "замок" - castle vs lock, "готов" - Goths vs ready)
- Make speech sound unnatural
- Reduce TTS intelligibility

While Silero TTS produces high-quality Russian speech, it relies on correct stress placement. Without explicit stress markers, homographs and uncommon words may be mispronounced.

### Goal

Integrate the [Silero Stress](https://github.com/snakers4/silero-stress) library via a new REST service (`biblio-stress-server-silero`) to automatically add stress markers (`+`) to Russian text before TTS synthesis.

### Solution

Add a stress marking step in the text processing pipeline between normalization and SSML processing.

### Integration Point

The stress marking should be applied **after text normalization but before SSML processing and chunking**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    ABB-TTS Text Processing                       │
│                                                                  │
│   1. Parse eBook (EPUB/FB2)                                     │
│      ↓                                                           │
│   2. Normalize text (numbers → words, cleanup)                  │
│      ↓                                                           │
│   3. ★ STRESS MARKING (if Russian + enabled) ★                  │
│      │   POST /stress-silero/api/stress                         │
│      │   Input: "Я готов открыть замок"                         │
│      │   Output: "+Я гот+ов откр+ыть зам+ок"                    │
│      ↓                                                           │
│   4. Add SSML break tags (sentence/paragraph pauses)            │
│      ↓                                                           │
│   5. Chunk text (respecting max chunk size)                     │
│      ↓                                                           │
│   6. Send to TTS (Silero TTS)                                   │
│      ↓                                                           │
│   7. Generate audio                                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Implementation Plan

**Phase 1: Stress Client**
1. ⏳ Create `internal/stress/` package with:
   - `client.go` - HTTP client for stress server
   - `models.go` - Request/response models
2. ⏳ Add stress configuration to `internal/config/config.go`

**Phase 2: Pipeline Integration**
3. ⏳ Integrate stress client into `internal/tts/adapter.go`
4. ⏳ Add language detection for automatic Russian text handling
5. ⏳ Add stress marking toggle to provider settings

**Phase 3: UI Integration**
6. ⏳ Add stress marking toggle to TTS Settings tab
7. ⏳ Add stress server URL configuration
8. ⏳ Display stress server status in provider list

### Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `stress_marking_enabled` | `true` | Enable stress marking for Russian text |
| `stress_server_url` | `http://stress-silero:80/stress-silero` | Stress server URL |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ABB_TTS_STRESS_ENABLED` | Enable stress marking | `true` |
| `ABB_TTS_STRESS_SERVER_URL` | Stress server URL | `http://stress-silero:80/stress-silero` |

### Language Detection

Stress marking is automatically applied when:
1. `stress_marking_enabled` is `true`
2. The selected TTS voice language is Russian (`ru`)
3. The stress server is available (health check passes)

### Files to Modify

- `internal/config/config.go` - Add stress configuration
- `internal/stress/client.go` - New stress client package
- `internal/stress/models.go` - Request/response models
- `internal/tts/adapter.go` - Integrate stress marking into pipeline
- `internal/server/settings.go` - Add UI settings
- `internal/server/templates/index.html` - Add stress toggle to UI
- `internal/server/assets/app.js` - Handle stress settings

### Related Components

- **Stress Server**: [biblio-stress-server-silero](https://github.com/vpoluyaktov/biblio-stress-server-silero)
- **TTS Server**: [biblio-tts-server-silero](https://github.com/vpoluyaktov/biblio-tts-server-silero)

**Date:** 2026-02-02

---

*Last updated: 2026-02-02*
