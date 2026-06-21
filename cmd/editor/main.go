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
	"github.com/yourname/video-editor/internal/preset"
	"github.com/yourname/video-editor/internal/renderer"
	"github.com/yourname/video-editor/internal/subtitle"
	"github.com/yourname/video-editor/internal/transcription"
	"github.com/yourname/video-editor/internal/web"
)

func main() {
	// Standard input dimensions (1920x1080 for most streamers)
	// These can be made configurable from preset later
	inputWidth := 1920
	inputHeight := 1080

	// Define command line flags
	videosDirFlag := flag.String("dir", "videos", "Directory containing input videos")
	urlFlag := flag.String("url", "", "YouTube URL to download and process")
	layoutsFlag := flag.Bool("layouts", false, "Choose from exported layouts and bypass Web UI")
	layoutEditFlag := flag.Bool("layoutedit", false, "Bypass layout edit prompt and edit in Web UI")
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

	useLayoutsFlag := *layoutsFlag
	useLayoutEditFlag := *layoutEditFlag

	for _, arg := range flag.Args() {
		if arg == "-layouts" || arg == "--layouts" {
			useLayoutsFlag = true
		}
		if arg == "-layoutedit" || arg == "--layoutedit" {
			useLayoutEditFlag = true
		}
	}

	alreadyDownloaded := false
	alreadyProcessed := false
	var downloadedFile string
	if youtubeURL != "" {
		downloadedFile = getYTFilename(youtubeURL, `C:\Users\s\downloads\cookies.txt`)
		if downloadedFile != "" && fileExists(downloadedFile) {
			alreadyDownloaded = true
			baseName := filepath.Base(downloadedFile)
			finalOutputPath := filepath.Join("output", baseName)
			alreadyProcessed = fileExists(finalOutputPath)
		}
	}

	// Prompt the user to opt out of rendering subtitles at the start of the script
	skipSubtitles := false
	if alreadyDownloaded && alreadyProcessed {
		fmt.Print("Video is already downloaded and processed. Do you want to process without adding subtitles? (y/n): ")
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		skipSubtitles = response == "y" || response == "yes"
	} else {
		fmt.Print("Do you want to render subtitles? (y/n): ")
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		skipSubtitles = response == "n" || response == "no"
	}

	var videoFiles []string

	if youtubeURL != "" {
		if alreadyDownloaded {
			videoFiles = []string{downloadedFile}
			log.Printf("Video is already downloaded: %s. Skipping download.\n", downloadedFile)
		} else {
			log.Printf("Downloading video from URL: %s\n", youtubeURL)

			if err := os.MkdirAll("videos", 0755); err != nil {
				log.Fatalf("Failed to create videos directory: %v", err)
			}

			var downloadSuccess bool
			cookiesPath := `C:\Users\s\downloads\cookies.txt`

			// Attempt 1: Static cookies file if it exists
			if fileExists(cookiesPath) {
				cmd := exec.Command("yt-dlp",
					"--cookies", cookiesPath,
					"--js-runtimes", "node",
					"-f", "299+140",
					"--merge-output-format", "mp4",
					"-o", "videos/%(title)s.%(ext)s",
					youtubeURL,
				)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				log.Printf("Executing: %s\n", strings.Join(cmd.Args, " "))
				if err := cmd.Run(); err == nil {
					downloadSuccess = true
				} else {
					log.Printf("Download with static cookies failed: %v\n", err)
				}
			} else {
				log.Printf("Static cookies file not found at %s. Skipping to browser cookies...\n", cookiesPath)
			}

			// Attempt 2: Fallback to chrome cookies
			if !downloadSuccess {
				cmd := exec.Command("yt-dlp",
					"--cookies-from-browser", "chrome",
					"--js-runtimes", "node",
					"-f", "299+140",
					"--merge-output-format", "mp4",
					"-o", "videos/%(title)s.%(ext)s",
					youtubeURL,
				)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				log.Printf("Executing: %s\n", strings.Join(cmd.Args, " "))
				if err := cmd.Run(); err == nil {
					downloadSuccess = true
				} else {
					log.Printf("Download with chrome cookies failed: %v\n", err)
				}
			}

			// Attempt 3: Fallback to edge cookies
			if !downloadSuccess {
				cmd := exec.Command("yt-dlp",
					"--cookies-from-browser", "edge",
					"--js-runtimes", "node",
					"-f", "299+140",
					"--merge-output-format", "mp4",
					"-o", "videos/%(title)s.%(ext)s",
					youtubeURL,
				)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				log.Printf("Executing: %s\n", strings.Join(cmd.Args, " "))
				if err := cmd.Run(); err == nil {
					downloadSuccess = true
				} else {
					log.Printf("Download with edge cookies failed: %v\n", err)
				}
			}

			// Attempt 4: Fallback to no cookies
			if !downloadSuccess {
				cmd := exec.Command("yt-dlp",
					"--js-runtimes", "node",
					"-f", "299+140",
					"--merge-output-format", "mp4",
					"-o", "videos/%(title)s.%(ext)s",
					youtubeURL,
				)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				log.Printf("Executing: %s\n", strings.Join(cmd.Args, " "))
				if err := cmd.Run(); err == nil {
					downloadSuccess = true
				} else {
					log.Printf("\n[Error] Failed to download video using yt-dlp.")
					log.Printf("If this is an age-restricted video, please make sure you have exported a valid cookies file from your logged-in browser to %s.", cookiesPath)
					log.Printf("You can export cookies using browser extensions like 'Get cookies.txt LOCALLY' or 'cookie.txt'.\n")
					log.Fatalf("All download attempts failed. Last error: %v", err)
				}
			}

			// Find the downloaded video
			downloadedFile = findNewestVideo("videos")
			if downloadedFile == "" {
				log.Fatalf("Could not find downloaded video file in videos directory")
			}

			videoFiles = []string{downloadedFile}
			log.Printf("Downloaded video to: %s\n", downloadedFile)
		}
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
		transcriptSRTPath := filepath.Join("output", inputBaseNoExt+".srt")
		assFilePath := filepath.Join("output", inputBaseNoExt+".ass")

		if !skipSubtitles {

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

		// --- Visual Layout Editing Option ---
		presetPath := "presets/shorts.json"
		bypassWebUI := false

		if useLayoutsFlag {
			layoutsDir := "layouts"
			files, err := os.ReadDir(layoutsDir)
			if err != nil || len(files) == 0 {
				log.Printf("No layouts found in %s/ directory. Proceeding with default.\n", layoutsDir)
			} else {
				var jsonFiles []string
				for _, f := range files {
					if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".json") {
						jsonFiles = append(jsonFiles, f.Name())
					}
				}

				if len(jsonFiles) == 0 {
					log.Printf("No .json layouts found in %s/ directory. Proceeding with default.\n", layoutsDir)
				} else {
					var selectedFile string
					if len(jsonFiles) == 1 {
						selectedFile = jsonFiles[0]
						log.Printf("Found 1 layout: %s. Using it.\n", selectedFile)
					} else {
						fmt.Println("\nAvailable layouts:")
						for idx, name := range jsonFiles {
							fmt.Printf("  %d. %s\n", idx+1, name)
						}
						fmt.Print("Select a layout (1-", len(jsonFiles), "): ")
						var choice int
						_, scanErr := fmt.Scanln(&choice)
						if scanErr != nil || choice < 1 || choice > len(jsonFiles) {
							log.Println("Invalid choice. Using default layout.")
						} else {
							selectedFile = jsonFiles[choice-1]
						}
					}

					if selectedFile != "" {
						srcPath := filepath.Join(layoutsDir, selectedFile)
						layoutData, err := os.ReadFile(srcPath)
						if err != nil {
							log.Printf("Failed to read layout file %s: %v\n", srcPath, err)
						} else {
							if err := os.WriteFile(presetPath, layoutData, 0644); err != nil {
								log.Printf("Failed to update preset file %s: %v\n", presetPath, err)
							} else {
								log.Printf("✓ Applied layout: %s\n", selectedFile)
								bypassWebUI = true
							}
						}
					}
				}
			}
		}

		startEditor := false
		if !bypassWebUI {
			if useLayoutEditFlag {
				startEditor = true
			} else {
				fmt.Print("Do you want to edit the layout visually in the browser? (y/n): ")
				var editResponse string
				fmt.Scanln(&editResponse)
				editResponse = strings.TrimSpace(strings.ToLower(editResponse))
				startEditor = (editResponse == "y" || editResponse == "yes")
			}
		}

		if startEditor {
			srv, err := web.NewServer(presetPath, videoInputPath, inputWidth, inputHeight, "output/preview.png", transcriptSRTPath, assFilePath)
			if err != nil {
				log.Fatalf("Failed to initialize Layout Editor server: %v", err)
			}
			if err := srv.Start(8080); err != nil {
				log.Fatalf("Failed to start Layout Editor server: %v", err)
			}

			fmt.Println("\n==================================================")
			fmt.Println("  Visual Layout Editor is running at:")
			fmt.Println("  http://localhost:8080")
			fmt.Println("==================================================")
			fmt.Println("  1. Open the URL in your browser.")
			fmt.Println("  2. Interactively adjust your layers.")
			fmt.Println("  3. Click 'Save Layout' in the UI when finished.")
			fmt.Println("  4. Press ENTER in this terminal to render the video...")
			fmt.Println("==================================================")

			// Wait for user to press enter
			var enterResp string
			fmt.Scanln(&enterResp)

			srv.Stop()
		}

		// Update subtitle ASS path if it was generated during editing session
		if fileExists(assFilePath) {
			assFileToUse = assFilePath
		}

		// Load/Reload the layout configurations from preset to be sure we have the latest
		p, err := preset.Load(presetPath)
		if err != nil {
			log.Fatalf("Failed to load preset %s: %v", presetPath, err)
		}
		concreteLayers := actions.NewLayersFromPreset(p, inputWidth, inputHeight)

		// Build dynamic input files list and map overlay layers
		inputArgs, activeLayers := actions.BuildRenderingGraph(concreteLayers, videoInputPath)

		graph := ffmpeg.BuildFilterGraph(activeLayers, assFileToUse)

		log.Printf("[%s] Starting video rendering...\n", baseName)
		err = renderer.Render(
			inputArgs,
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

func getYTFilename(youtubeURL string, cookiesPath string) string {
	// Attempt with static cookies
	if fileExists(cookiesPath) {
		cmd := exec.Command("yt-dlp",
			"--cookies", cookiesPath,
			"--js-runtimes", "node",
			"-f", "299+140",
			"--merge-output-format", "mp4",
			"-o", "videos/%(title)s.%(ext)s",
			"--get-filename",
			youtubeURL,
		)
		if out, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	// Fallback to chrome cookies
	cmdChrome := exec.Command("yt-dlp",
		"--cookies-from-browser", "chrome",
		"--js-runtimes", "node",
		"-f", "299+140",
		"--merge-output-format", "mp4",
		"-o", "videos/%(title)s.%(ext)s",
		"--get-filename",
		youtubeURL,
	)
	if out, err := cmdChrome.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	// Fallback to edge cookies
	cmdEdge := exec.Command("yt-dlp",
		"--cookies-from-browser", "edge",
		"--js-runtimes", "node",
		"-f", "299+140",
		"--merge-output-format", "mp4",
		"-o", "videos/%(title)s.%(ext)s",
		"--get-filename",
		youtubeURL,
	)
	if out, err := cmdEdge.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	// Fallback to no cookies
	cmdNoCookies := exec.Command("yt-dlp",
		"--js-runtimes", "node",
		"-f", "299+140",
		"--merge-output-format", "mp4",
		"-o", "videos/%(title)s.%(ext)s",
		"--get-filename",
		youtubeURL,
	)
	if out, err := cmdNoCookies.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}
