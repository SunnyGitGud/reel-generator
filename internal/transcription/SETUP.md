# Whisper Integration Setup

This guide will help you set up OpenAI's Whisper for audio transcription in the videdit project.

## Prerequisites

### 1. Install Python (if not already installed)
- Download from https://www.python.org/downloads/
- Make sure Python is added to your PATH
- Verify installation: `python --version`

### 2. Install Whisper
Run one of the following commands:

**Option A: Using pip (recommended)**
```bash
pip install openai-whisper
```

**Option B: With CUDA support (for GPU acceleration)**
```bash
pip install openai-whisper torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu118
```

### 3. Verify Installation
```bash
whisper --help
whisper --version
```

You should see the Whisper help text and version information.

## Model Sizes

Whisper comes with different model sizes. Choose based on your needs:

| Model | Size | Speed | Accuracy |
|-------|------|-------|----------|
| tiny | 39MB | Very Fast | Lower |
| base | 140MB | Fast | Good |
| small | 461MB | Medium | Better |
| medium | 1.4GB | Slow | Great |
| large | 2.9GB | Very Slow | Best |

First time using a model will download it (~0.5-3GB depending on model size).

## Usage in Go

```go
package main

import (
	"log"
	"github.com/yourname/video-editor/internal/transcription"
)

func main() {
	// Create options with desired settings
	opts := transcription.TranscriptionOptions{
		Model:     "base",
		Language:  "en",
		Device:    "cuda",        // or "cuda" for GPU
		TaskType:  "transcribe", // or "translate"
		OutputDir: "output",
	}

	// Transcribe audio file
	result, err := transcription.TranscribeAudio("videos/input.mp4", opts)
	if err != nil {
		log.Fatal(err)
	}

	// Get full transcript
	transcript := result.GetFullTranscript()
	log.Println("Transcript:", transcript)

	// Save transcript to file
	result.SaveToFile("output/transcript.txt")

	// Save with timestamps (SRT format)
	result.SaveSegmentsToSRT("output/transcript.srt")

	// Access segments with timing information
	for _, segment := range result.GetSegments() {
		log.Printf("[%.2f - %.2f] %s\n", segment.Start, segment.End, segment.Text)
	}

	// Get text at specific timestamp
	seg := result.GetSegmentByTime(5.5)
	if seg != nil {
		log.Printf("At 5.5s: %s\n", seg.Text)
	}
}
```

## Troubleshooting

### Command not found: whisper
- Verify Python is installed and accessible: `python --version`
- Verify Whisper is installed: `pip list | grep whisper`
- On Windows, you may need to restart your terminal after installation

### Memory errors with large models
- Use a smaller model size (e.g., "tiny" or "base")
- Or enable GPU support with CUDA

### CUDA/GPU Support Issues
1. Install NVIDIA CUDA Toolkit: https://developer.nvidia.com/cuda-downloads
2. Reinstall PyTorch with CUDA: `pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu118`
3. Set device to "cuda" in options

### Slow Performance on CPU
- Use a smaller model ("tiny" or "base")
- Enable GPU if available (CUDA)
- Consider using a cloud service instead

## Integration with FFmpeg

Since your project already uses FFmpeg, you can extract audio from video before transcription:

```go
// Extract audio from video using FFmpeg
cmd := exec.Command("ffmpeg", "-i", "videos/input.mp4", "-q:a", "9", "-n", "output/audio.mp3")
err := cmd.Run()

// Then transcribe the audio
result, err := transcription.TranscribeAudio("output/audio.mp3", opts)
```

## Language Support

Whisper supports 99+ languages. Some common codes:
- en = English
- es = Spanish
- fr = French
- de = German
- ja = Japanese
- zh = Chinese

See full list: https://github.com/openai/whisper/blob/main/whisper/tokenizer.py
