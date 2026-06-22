package audio

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type AnalysisResult struct {
	Duration   float64   `json:"duration"`
	IntervalMs int       `json:"interval_ms"`
	RMS        []float64 `json:"rms"`
	Peaks      []float64 `json:"peaks"`
	DB         []float64 `json:"db"`
}

func AnalyzeAudio(videoPath string, jsonPath string) (*AnalysisResult, error) {
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
		return nil, err
	}

	duration, err := getVideoDuration(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get video duration: %w", err)
	}

	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vn",
		"-af", "aresample=8000",
		"-f", "s16le",
		"-ac", "1",
		"-",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// 8000 samples per second.
	// 50ms interval = 8000 * 0.05 = 400 samples.
	// Each sample is 2 bytes (s16le).
	// So 400 samples = 800 bytes.
	const samplesPerChunk = 400
	const bytesPerChunk = samplesPerChunk * 2

	var rmsList []float64
	var peakList []float64
	var dbList []float64

	buf := make([]byte, bytesPerChunk)

	for {
		n, err := readFully(stdout, buf)
		if n == 0 {
			break
		}

		samplesRead := n / 2
		if samplesRead == 0 {
			break
		}

		sumSq := 0.0
		peak := 0.0

		for i := 0; i < samplesRead; i++ {
			low := buf[i*2]
			high := buf[i*2+1]
			val := int16(uint16(low) | (uint16(high) << 8))

			fVal := float64(val) / 32768.0

			sumSq += fVal * fVal
			absF := math.Abs(fVal)
			if absF > peak {
				peak = absF
			}
		}

		rms := math.Sqrt(sumSq / float64(samplesRead))
		db := -90.0
		if rms > 0 {
			db = 20.0 * math.Log10(rms)
			if db < -90.0 {
				db = -90.0
			}
		}

		rmsList = append(rmsList, rms)
		peakList = append(peakList, peak)
		dbList = append(dbList, db)

		if err != nil {
			break
		}
	}

	cmd.Wait()

	result := &AnalysisResult{
		Duration:   duration,
		IntervalMs: 50,
		RMS:        rmsList,
		Peaks:      peakList,
		DB:         dbList,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json: %w", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write json file: %w", err)
	}

	return result, nil
}

func readFully(r io.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		if n > 0 {
			total += n
		}
		if err != nil {
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}
	}
	return total, nil
}

func getVideoDuration(inputPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var duration float64
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &duration)
	if err != nil {
		return 0, err
	}
	return duration, nil
}
