# Quick Start Guide - Whisper Transcription

## Installation

1. **Install Python** (if not already installed)
   ```bash
   python --version  # Verify Python 3.9+
   ```

2. **Install Whisper**
   ```bash
   pip install openai-whisper
   ```

3. **Verify Whisper**
   ```bash
   whisper --version
   ```

## Basic Usage

### Simple transcription
```bash
go run cmd/transcribe/main.go -file videos/input.mp4
```

### With custom model and language
```bash
go run cmd/transcribe/main.go -file videos/input.mp4 -model small -lang es
```

### Translate to English
```bash
go run cmd/transcribe/main.go -file videos/input.mp4 -task translate
```

### With GPU acceleration
```bash
go run cmd/transcribe/main.go -file audio.mp3 -device cuda -model medium
```

### Verbose output with all formats
```bash
go run cmd/transcribe/main.go -file videos/input.mp4 -format all -v
```

## Programmatic Usage

```go
package main

import (
	"log"
	"github.com/yourname/video-editor/internal/transcription"
)

func main() {
	opts := transcription.TranscriptionOptions{
		Model:     "base",
		Language:  "en",
		TaskType:  "transcribe",
		OutputDir: "output",
	}

	result, err := transcription.TranscribeAudio("videos/input.mp4", opts)
	if err != nil {
		log.Fatal(err)
	}

	// Use transcript
	println(result.GetFullTranscript())
	result.SaveToFile("output/transcript.txt")
}
```

## Integration with Video Rendering

```go
// In your main video processing pipeline:

// 1. Render video
graph := ffmpeg.BuildFilterGraph(layers)
renderer.Render("videos/input.mp4", "output/result.mp4", graph)

// 2. Transcribe audio from original video
opts := transcription.DefaultOptions()
result, err := transcription.TranscribeAudio("videos/input.mp4", opts)

// 3. Save transcript alongside video
result.SaveToFile("output/result_transcript.txt")
result.SaveSegmentsToSRT("output/result_subtitle.srt")
```

## Output Files

The transcription generates:

- **transcript.txt** - Full plain text transcript
- **transcript.srt** - SubRip subtitle format with timestamps
- **result.json** - Full Whisper output with segments and metadata

## Models and Performance

| Model | Download | Speed | Quality |
|-------|----------|-------|---------|
| tiny | 39MB | ⚡⚡⚡ | ⭐⭐ |
| base | 140MB | ⚡⚡ | ⭐⭐⭐ |
| small | 461MB | ⚡ | ⭐⭐⭐⭐ |
| medium | 1.4GB | 🐢 | ⭐⭐⭐⭐⭐ |
| large | 2.9GB | 🐢🐢 | ⭐⭐⭐⭐⭐ |

## API Reference

### TranscribeAudio(audioPath, options)
Transcribes an audio/video file and returns a TranscriptResult.

### TranscriptResult Methods
- `GetFullTranscript()` - Returns complete text
- `GetSegments()` - Returns all segments with timing
- `GetSegmentByTime(seconds)` - Get segment at specific time
- `SaveToFile(path)` - Save as plain text
- `SaveSegmentsToSRT(path)` - Save as SRT subtitles

## Troubleshooting

**Issue: "command not found: whisper"**
- Check: `where whisper` or `which whisper`
- Solution: Reinstall and restart terminal

**Issue: Out of memory**
- Solution: Use smaller model (`tiny` or `base`)
- Or enable CUDA for GPU processing

**Issue: Very slow**
- Solution: Enable GPU with CUDA
- Or use smaller model

## Next Steps

1. Run the transcription CLI: `go run cmd/transcribe/main.go -file videos/input.mp4`
2. Check the generated files in `output/`
3. Integrate into your main editor workflow as needed
