# Biblio Audiobook Builder TTS

> Part of the [BiblioHub](https://github.com/vpoluyaktov/BiblioHub) application suite

## Description

Biblio Audiobook Builder TTS is a powerful tool that converts electronic books in .epub and .fb2 formats into audiobooks using text-to-speech technology. It supports both local and cloud-based TTS services, allowing you to create high-quality audiobooks from your digital library.

## Features
- TUI interface for easy interaction
- Support for .epub and .fb2 book formats
- Multiple TTS providers (local and cloud-based)
- Customizable voice selection
- Adjustable speech parameters (speed, pitch)
- Audiobook metadata management
- M4B audiobook format output
- Integration with Audiobookshelf server

## Installation

### Prerequisites

To use Audiobook Builder TTS, you need to have the following utilities installed:

- **ffmpeg** (for audio processing)
- **ffprobe** (for audio metadata)

For Linux:
```bash
sudo apt install ffmpeg
```

For MacOS (using Homebrew):
```bash
brew install ffmpeg
```

For Windows, visit [ffmpeg website](https://ffmpeg.org/download.html) for installation instructions.

## Build Instructions

1. Clone the repository:
   ```bash
   git clone https://github.com/vpoluyaktov/biblio-audiobook-builder-tts.git
   ```

2. Ensure Go is installed on your system

3. Build the application:
   ```bash
   cd biblio-audiobook-builder-tts
   go build
   ```

## Configuration

The application can be configured using the `biblio-audiobook-builder-tts.config.yaml` file. Key configuration options include:

- Log file location
- Output directory for audiobooks
- Temporary processing directory
- Default TTS voice
- Default TTS provider

## Usage

1. Run the application:
   ```bash
   ./biblio-audiobook-builder-tts
   ```

2. Follow the TUI interface to:
   - Select an input book file
   - Choose TTS voice and provider
   - Adjust speech parameters
   - Generate the audiobook

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
