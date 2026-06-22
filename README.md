# Videdit - Reel Generator & Layout Editor

Videdit is a powerful Go-based vertical video creator and layout editor designed to format horizontal videos (such as Twitch streams or YouTube videos) into vertical clips optimized for TikTok, YouTube Shorts, and Instagram Reels. 

The application features a visual, web-based layout editor with layers, audio waveforms, automated silence detection, manual cut editing, and customizable subtitles.

---

## Features

### 1. Web App Landing Page
- Simple SPA UI to paste YouTube URLs or local video paths.
- Toggle switch to include/exclude Whisper audio transcription (Subtitles) for faster initialization.
- Real-time download and transcription progress loader.

### 2. Visual Layout Editor
- **Multi-Layer Composition**: Add and customize layers (`Background`, `Gameplay`, `Facecam`, `Overlay`, and custom additional layers).
- **Interactive UI**: Click and drag layers to reposition, resize via handles, rename, lock/unlock, toggle visibility, and adjust z-index rendering order.
- **Scale Modes**: Supports `Stretch`, `Fit (Contain)`, and `Fill (Cover)` scaling with aspect-ratio locking.
- **Dynamic Previews**: Instant interactive layer layouts with background-generated subtitle and preview frame generation.

### 3. Smart Waveform Timeline & Cutting
- **Waveform Visualization**: Draws the audio waveform in real time along with cached frame thumbnails.
- **Timeline Zoom**: Zoom in/out on the timeline track (up to 10.0x) with scroll-aware ticks.
- **Auto-Cut Silence Detection**: Remove silent sections automatically based on configurable DB threshold, padding, and minimum duration presets (Very Conservative to Very Aggressive).
- **Premiere-style Manual Cuts**: Split segments at the playhead (Shortcut: `S` or `+ Cut`), drag boundaries, toggle/disable specific clips, delete clips, and reorder clips via drag-and-drop.

### 4. Custom Subtitles & Overlap Resolution
- Customize subtitle styles (font family, font size, vertical margin, primary color, outline color, and 9-grid text alignments).
- Real-time subtitle editor card list with text areas and timestamp fine-tuning.
- Automated subtitle shift and overlap resolution to ensure captions align perfectly with split/trimmed video segments.
- Renders and burns subtitles onto the final composition using FFmpeg's `ass` filter post-concatenation.

---

## Build Requirements

- **Go**: Version `1.25.1` (as specified in `go.mod`)
- **Python**: Version `3.10+` (needed for Whisper transcription and romanization)
- **External Binaries** (Must be added to system `PATH`):
  - **FFmpeg & FFprobe**: Version `5.0+` or `6.0+` (compiled with `libx264`, `aac`, and Nvidia `h264_nvenc` support if hardware acceleration is desired).
  - **yt-dlp**: Latest release (for fetching YouTube videos).
  - **Whisper CLI**: Python `openai-whisper` package (usually installed via `pip install openai-whisper`).

---

## Commands & Usage

### 1. Visual Layout Editor (`cmd/editor`)
This is the main web server app.

- **Run in Daemon Mode** (Default web app):
  ```bash
  go run ./cmd/editor
  ```
  Open your browser to `http://localhost:8080` to access the editor.

- **Pre-fill with Video Path or YouTube URL**:
  ```bash
  go run ./cmd/editor "https://www.youtube.com/watch?v=VIDEO_ID"
  # OR
  go run ./cmd/editor "videos/sample.mp4"
  ```

- **Run in Terminal Interactive TUI Mode**:
  ```bash
  go run ./cmd/editor -tui
  ```

- **All Configuration Flags**:
  - `-dir <path>`: Directory containing input videos (default: `videos`).
  - `-url <youtube_url>`: YouTube URL to download and process.
  - `-layouts`: Choose from exported layouts and bypass the Web UI.
  - `-layoutedit`: Bypass the layout edit prompt and edit in the Web UI.
  - `-tui`: Run in terminal interactive mode instead of daemon server.

---

### 2. Audio/Video Transcription CLI (`cmd/transcribe`)
Transcribe audio/video files to text and subtitle files using Whisper directly from the CLI.

- **Usage**:
  ```bash
  go run ./cmd/transcribe -file <path_to_video_or_audio> [options]
  ```

- **Options**:
  - `-model <type>`: Whisper model size: `tiny`, `base`, `small`, `medium`, `large` (default: `base`).
  - `-lang <lang_code>`: Target language code, e.g., `en`, `es`, `de` (default: `en`).
  - `-output <path>`: Output directory for transcripts (default: `output`).
  - `-task <type>`: Task type: `transcribe` or `translate` (default: `transcribe`).
  - `-device <type>`: Execution device: `cpu` or `cuda` (default: `cuda`).
  - `-format <format>`: Output format: `txt`, `srt`, or `all` (default: `all`).
  - `-romanize`: If passed, romanizes Devanagari Hindi text to Hinglish.

---

### 3. Subtitle Management CLI (`cmd/subtitle`)
Parse, style, burn, or print subtitle metadata directly from the command line.

- **Parse Subtitles**:
  ```bash
  go run ./cmd/subtitle parse -file output/transcript.srt [-v]
  ```
- **Apply Style Template to SRT and Generate ASS**:
  ```bash
  go run ./cmd/subtitle style -file output/transcript.srt -style <style> -output output/subs.ass
  # Styles available: tiktok, youtube, instagram, classic
  ```
- **Burn Subtitles onto Video**:
  ```bash
  go run ./cmd/subtitle burn -input videos/input.mp4 -ass output/subs.ass -output output.mp4
  ```
- **Get SRT Metadata Info**:
  ```bash
  go run ./cmd/subtitle info -file output/transcript.srt
  ```
