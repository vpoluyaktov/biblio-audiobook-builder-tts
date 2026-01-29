# Biblio Audiobook Builder TTS

> Part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) application suite

## Description

Biblio Audiobook Builder TTS is a server-based application that converts e-books (EPUB, FB2) into audiobooks using text-to-speech technology. It provides a web interface for uploading books, monitoring conversion progress, and downloading completed audiobooks.

## Features

- **Web Interface**: Upload books, monitor progress, download audiobooks
- **Drop-and-Forget**: Start conversion, close browser, reconnect later
- **Multiple TTS Engines**: Local (eSpeak), cloud (Google, OpenAI), self-hosted (Silero, OpenVoice)
- **EPUB and FB2 Support**: Parse and convert popular e-book formats
- **M4B Output**: Audiobooks with chapter markers, metadata, and cover art
- **OPDS Integration**: Browse and convert books from Biblio Catalog
- **Audiobookshelf Integration**: Auto-upload completed audiobooks
- **Real-time Progress**: WebSocket-based live updates

## Quick Start (Docker)

The recommended way to run Audiobook Builder TTS is as part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) Docker Swarm stack:

```bash
# Clone BiblioHub and all service repositories
git clone https://github.com/vpoluyaktov/biblio-hub.git
git clone https://github.com/vpoluyaktov/biblio-audiobook-builder-tts.git
git clone https://github.com/vpoluyaktov/biblio-tts-server-silero.git
git clone https://github.com/vpoluyaktov/biblio-ebooks-catalog.git

# Start the stack
cd biblio-hub
cp .env.example .env
./scripts/start_stack.sh
```

Access at: `http://localhost:9900/abb-tts/`

## Standalone Installation

### Prerequisites

- **Go 1.21+**
- **ffmpeg** and **ffprobe** (for audio processing)

```bash
# Linux
sudo apt install ffmpeg

# macOS
brew install ffmpeg
```

### Build and Run

```bash
git clone https://github.com/vpoluyaktov/biblio-audiobook-builder-tts.git
cd biblio-audiobook-builder-tts
go build
./biblio-audiobook-builder-tts
```

The web interface will open at `http://localhost:9901`

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ABB_TTS_HOST` | Server host | `0.0.0.0` |
| `ABB_TTS_PORT` | Server port | `9901` |
| `ABB_TTS_BASE_PATH` | URL base path (for reverse proxy) | `/` |
| `ABB_TTS_SERVER_URL` | Silero TTS server URL | - |
| `ABB_TTS_OPDS_SERVER_URL` | Biblio Catalog URL | - |

### Command Line Flags

```bash
./biblio-audiobook-builder-tts --port 8080 --no-browser --log-level DEBUG
```

## Documentation

See [Specification.md](Specification.md) for detailed technical documentation including:
- REST API reference
- TTS provider configuration
- Project structure
- Docker deployment details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
