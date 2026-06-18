package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yourname/video-editor/internal/actions"
	"github.com/yourname/video-editor/internal/ffmpeg"
	"github.com/yourname/video-editor/internal/renderer"
	"github.com/yourname/video-editor/internal/subtitle"
	"github.com/yourname/video-editor/internal/transcription"
)

func main() {
	// Standard input dimensions (1920x1080 for most streamers)
	// These can be made configurable from preset later
	inputWidth := 1920
	inputHeight := 1080

	// Define command line flags
	videosDirFlag := flag.String("dir", "videos", "Directory containing input videos")
	urlFlag := flag.String("url", "", "YouTube URL to download and process")
	flag.Parse()

	videosDir := *videosDirFlag

	// Detect if a URL was passed via flag or positional args
	youtubeURL := *urlFlag
	if youtubeURL == "" {
		for _, arg := range flag.Args() {
			if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
				youtubeURL = arg
				break
			}
		}
	}

	// Prompt the user to opt out of rendering subtitles at the start of the script
	fmt.Print("Do you want to render subtitles? (y/n): ")
	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))
	skipSubtitles := response == "n" || response == "no"

	var videoFiles []string

	if youtubeURL != "" {
		log.Printf("Downloading video from URL: %s\n", youtubeURL)

		if err := os.MkdirAll("videos", 0755); err != nil {
			log.Fatalf("Failed to create videos directory: %v", err)
		}

		// Get the filename that yt-dlp will produce
		cmdName := exec.Command("yt-dlp",
			"--cookies", `C:\Users\s\downloads\cookies.txt`,
			"--js-runtimes", "node",
			"-f", "299+140",
			"--merge-output-format", "mp4",
			"-o", "videos/%(title)s.%(ext)s",
			"--get-filename",
			youtubeURL,
		)
		nameOut, err := cmdName.Output()
		var downloadedFile string
		if err == nil {
			downloadedFile = strings.TrimSpace(string(nameOut))
		}

		// Run the actual download command
		cmd := exec.Command("yt-dlp",
			"--cookies", `C:\Users\s\downloads\cookies.txt`,
			"--js-runtimes", "node",
			"-f", "299+140",
			"--merge-output-format", "mp4",
			"-o", "videos/%(title)s.%(ext)s",
			youtubeURL,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Printf("Executing: %s\n", strings.Join(cmd.Args, " "))
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to download video: %v", err)
		}

		// If we couldn't get the filename before, or if it doesn't exist, let's find the newest video in videos/
		if downloadedFile == "" || !fileExists(downloadedFile) {
			downloadedFile = findNewestVideo("videos")
		}

		if downloadedFile == "" {
			log.Fatalf("Could not find downloaded video file in videos directory")
		}

		videoFiles = []string{downloadedFile}
		log.Printf("Downloaded video to: %s\n", downloadedFile)
	} else {
		// Read files in the directory
		files, err := os.ReadDir(videosDir)
		if err != nil {
			log.Fatalf("Failed to read videos directory %s: %v", videosDir, err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if ext == ".mp4" || ext == ".mov" || ext == ".mkv" || ext == ".avi" {
				videoFiles = append(videoFiles, filepath.Join(videosDir, file.Name()))
			}
		}
	}

	if len(videoFiles) == 0 {
		log.Printf("No video files found in directory: %s\n", videosDir)
		return
	}

	log.Printf("Found %d video(s) to process:\n", len(videoFiles))
	for _, vf := range videoFiles {
		log.Printf("  - %s\n", vf)
	}

	// Ensure output directory exists
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Process each video
	for _, videoInputPath := range videoFiles {
		log.Printf("\n========================================")
		log.Printf("Processing video: %s", videoInputPath)
		log.Printf("========================================\n")

		baseName := filepath.Base(videoInputPath)
		inputBaseNoExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		finalOutputPath := filepath.Join("output", baseName)

		var err error
		var assFileToUse string

		if !skipSubtitles {
			transcriptSRTPath := filepath.Join("output", inputBaseNoExt+".srt")
			assFilePath := filepath.Join("output", inputBaseNoExt+".ass")

			useManualSubs := false
			useSRT := false

			// Check if render output already exists
			if _, err := os.Stat(finalOutputPath); err == nil {
				fmt.Printf("\nRender output %s already exists.\nDo you want to render with manually edited subtitles? (y/N): ", finalOutputPath)
				var response string
				fmt.Scanln(&response)
				response = strings.TrimSpace(strings.ToLower(response))
				if response == "y" || response == "yes" {
					fmt.Print("Did you edit the [s]rt file or the [a]ss file? (s/a): ")
					var typeResponse string
					fmt.Scanln(&typeResponse)
					typeResponse = strings.TrimSpace(strings.ToLower(typeResponse))
					if typeResponse == "s" || typeResponse == "srt" {
						if _, err := os.Stat(transcriptSRTPath); err != nil {
							log.Printf("SRT file %s does not exist. Cannot use manual SRT. Proceeding with fresh transcription.\n", transcriptSRTPath)
						} else {
							useManualSubs = true
							useSRT = true
						}
					} else if typeResponse == "a" || typeResponse == "ass" {
						if _, err := os.Stat(assFilePath); err != nil {
							log.Printf("ASS file %s does not exist. Cannot use manual ASS. Proceeding with fresh transcription.\n", assFilePath)
						} else {
							useManualSubs = true
							useSRT = false
						}
					} else {
						log.Printf("Invalid option: %s. Proceeding with fresh transcription.\n", typeResponse)
					}
				}
			}

			if useManualSubs {
				if useSRT {
					log.Printf("[%s] Using manually edited SRT: %s\n", baseName, transcriptSRTPath)
					subtitleLayer := subtitle.NewSubtitleLayerWithDefaults(transcriptSRTPath)
					err = subtitleLayer.Prepare("output")
					if err != nil {
						log.Printf("[%s] Failed to prepare subtitle layer from manual SRT: %v\n", baseName, err)
						continue
					}
					assFileToUse = subtitleLayer.GetASSFile()
				} else {
					log.Printf("[%s] Using manually edited ASS directly: %s\n", baseName, assFilePath)
					assFileToUse = assFilePath
				}
			} else {
				// --- Transcription Step ---
				opts := transcription.DefaultOptions()
				opts.Model = "small" // Using small model for better quality
				opts.Language = "en"
				opts.OutputDir = "output"

				log.Printf("[%s] Starting audio transcription...\n", baseName)
				transcriptResult, err := transcription.TranscribeAudio(videoInputPath, opts)
				if err != nil {
					log.Printf("[%s] Transcription failed: %v\n", baseName, err)
					continue
				}

				// Save transcript to SRT for subtitles
				err = transcriptResult.SaveSegmentsToSRT(transcriptSRTPath)
				if err != nil {
					log.Printf("[%s] Failed to save SRT: %v\n", baseName, err)
					continue
				}
				log.Printf("[%s] Transcript saved to %s\n", baseName, transcriptSRTPath)

				// --- Subtitle Layer Setup ---
				subtitleLayer := subtitle.NewSubtitleLayerWithDefaults(transcriptSRTPath)

				// Prepare the subtitle layer (generates .ass file)
				err = subtitleLayer.Prepare("output")
				if err != nil {
					log.Printf("[%s] Failed to prepare subtitle layer: %v\n", baseName, err)
					continue
				}
				log.Printf("[%s] Generated ASS file: %s\n", baseName, subtitleLayer.GetASSFile())
				assFileToUse = subtitleLayer.GetASSFile()
			}
		}

		// --- Video Rendering with Subtitles ---
		concreteLayers := actions.NewLayersWithDimensions(inputWidth, inputHeight)

		layers := make([]ffmpeg.RenderLayer, len(concreteLayers))
		for i, layer := range concreteLayers {
			layers[i] = layer
		}

		graph := ffmpeg.BuildFilterGraph(layers, assFileToUse)

		log.Printf("[%s] Starting video rendering...\n", baseName)
		err = renderer.Render(
			videoInputPath,
			finalOutputPath,
			graph,
		)
		if err != nil {
			log.Printf("[%s] Rendering failed: %v\n", baseName, err)
			continue
		}

		log.Printf("[%s] Video rendered successfully to %s\n", baseName, finalOutputPath)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func findNewestVideo(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var newestFile string
	var newestTime os.FileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".mp4" || ext == ".mov" || ext == ".mkv" || ext == ".avi" {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if newestTime == nil || info.ModTime().After(newestTime.ModTime()) {
				newestTime = info
				newestFile = filepath.Join(dir, file.Name())
			}
		}
	}
	return newestFile
}
