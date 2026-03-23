# Biblio Audiobook Builder TTS

> Part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) application suite

![Biblio Audiobook Builder TTS](docs/images/abb-tts-screenshot.png)

Server-based application that converts e-books (EPUB, FB2) into audiobooks using text-to-speech technology. Provides a web interface for uploading books, monitoring conversion progress, and downloading completed audiobooks. Written in Go with a vanilla JavaScript frontend.

**Live demo: [https://demo.bibliohub.org/abb-tts/](https://demo.bibliohub.org/abb-tts/)**

## Features

- **Web Interface** — Upload books, monitor progress, download audiobooks
- **Drop-and-Forget** — Start conversion, close browser, reconnect later
- **Multiple TTS Engines** — Local (eSpeak), cloud (Google, OpenAI), self-hosted (Silero, OpenVoice, Piper)
- **EPUB and FB2 Support** — Parse and convert popular e-book formats
- **M4B Output** — Audiobooks with chapter markers, metadata, and cover art
- **OPDS Integration** — Browse and convert books from Biblio Catalog
- **Audiobookshelf Integration** — Auto-upload completed audiobooks
- **Text Preprocessing** — Abbreviation normalization, pronunciation dictionaries, Russian stress marking
- **Real-time Progress** — WebSocket-based live updates

## Technology Stack

- **Language**: Go 1.24+
- **Database**: SQLite
- **Frontend**: Vanilla JavaScript, HTML templates
- **Audio**: FFmpeg / FFprobe
- **E-book Parsing**: [biblio-ebook-parser](https://github.com/vpoluyaktov/biblio-ebook-parser) library
- **Deployment**: Docker, Docker Swarm (via BiblioHub)

## Quick Start (Docker)

The recommended way to run is as part of the [BiblioHub](https://github.com/vpoluyaktov/biblio-hub) Docker Swarm stack:

```bash
git clone https://github.com/vpoluyaktov/biblio-hub.git
cd biblio-hub
cp .env.example .env
./scripts/start_stack.sh
```

Access at: `http://localhost:9900/abb-tts/`

### From Source

```bash
git clone https://github.com/vpoluyaktov/biblio-audiobook-builder-tts.git
cd biblio-audiobook-builder-tts
go build
./biblio-audiobook-builder-tts
```

Requires Go 1.24+, FFmpeg, and FFprobe. The web interface opens at `http://localhost:9901`.

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `ABB_TTS_HOST` | Server host | `0.0.0.0` |
| `ABB_TTS_PORT` | Server port | `9901` |
| `ABB_TTS_BASE_PATH` | URL base path (for reverse proxy) | `/` |
| `ABB_TTS_SERVER_URL` | Silero TTS server URL | — |
| `ABB_TTS_OPDS_SERVER_URL` | Biblio Catalog URL | — |

## License

MIT
