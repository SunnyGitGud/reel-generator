package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yourname/video-editor/internal/actions"
	"github.com/yourname/video-editor/internal/ffmpeg"
	"github.com/yourname/video-editor/internal/preset"
	"github.com/yourname/video-editor/internal/preview"
	"github.com/yourname/video-editor/internal/subtitle"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	presetPath     string
	preset         *preset.Preset
	inputVideo     string
	inputWidth     int
	inputHeight    int
	layers         []actions.RenderLayer
	previewService *preview.PreviewService
	previewPath    string
	srtPath        string
	assPath        string
	currentTime    float64
	mu             sync.Mutex
	httpServer     *http.Server
}

type LayerResponse struct {
	Name              string  `json:"name"`
	X                 int     `json:"x"`
	Y                 int     `json:"y"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	Visible           bool    `json:"visible"`
	ZIndex            int     `json:"z_index"`
	ScaleMode         string  `json:"scale_mode"`
	CropXPercent      float64 `json:"crop_x_percent"`
	CropYPercent      float64 `json:"crop_y_percent"`
	CropWidthPercent  float64 `json:"crop_width_percent"`
	CropHeightPercent float64 `json:"crop_height_percent"`
	InputWidth        int     `json:"input_width"`
	InputHeight       int     `json:"input_height"`
}

type LayoutResponse struct {
	Width  int             `json:"width"`
	Height int             `json:"height"`
	Layers []LayerResponse `json:"layers"`
}

type PositionRequest struct {
	Name              string  `json:"name"`
	X                 int     `json:"x"`
	Y                 int     `json:"y"`
	Visible           bool    `json:"visible"`
	ZIndex            int     `json:"z_index"`
	CropXPercent      float64 `json:"crop_x_percent"`
	CropYPercent      float64 `json:"crop_y_percent"`
	CropWidthPercent  float64 `json:"crop_width_percent"`
	CropHeightPercent float64 `json:"crop_height_percent"`
}

type SizeRequest struct {
	Name      string `json:"name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	ScaleMode string `json:"scale_mode"`
}

type ExportRequest struct {
	Name string `json:"name"`
}

func NewServer(presetPath string, inputVideo string, inputWidth, inputHeight int, previewPath string, srtPath string, assPath string) (*Server, error) {
	p, err := preset.Load(presetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load preset: %w", err)
	}

	concreteLayers := actions.NewLayersFromPreset(p, inputWidth, inputHeight)
	previewService := preview.NewPreviewService(inputVideo)

	return &Server{
		presetPath:     presetPath,
		preset:         p,
		inputVideo:     inputVideo,
		inputWidth:     inputWidth,
		inputHeight:    inputHeight,
		layers:         concreteLayers,
		previewService: previewService,
		previewPath:    previewPath,
		srtPath:        srtPath,
		assPath:        assPath,
	}, nil
}

func (s *Server) Start(port int) error {
	// Generate initial preview image before starting server
	log.Println("Generating initial preview image...")
	if err := s.generatePreviewFile(); err != nil {
		log.Printf("Warning: failed to generate initial preview: %v\n", err)
	}

	log.Println("Generating raw reference frame image...")
	if err := s.generateRawFrameFile(); err != nil {
		log.Printf("Warning: failed to generate raw reference frame: %v\n", err)
	}

	mux := http.NewServeMux()

	// Serve index.html at root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := staticFS.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	})

	// Serve other static files
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))

	// API Endpoints
	mux.HandleFunc("/api/layout", s.handleGetLayout)
	mux.HandleFunc("/api/layout/position", s.handleUpdatePosition)
	mux.HandleFunc("/api/layout/size", s.handleUpdateSize)
	mux.HandleFunc("/api/layout/preview", s.handleRegeneratePreview)
	mux.HandleFunc("/api/layout/save", s.handleSaveLayout)
	mux.HandleFunc("/api/layout/export", s.handleExportLayout)
	mux.HandleFunc("/api/subtitles", s.handleGetSubtitles)
	mux.HandleFunc("/api/subtitles/style", s.handleUpdateSubtitlesStyle)
	mux.HandleFunc("/api/subtitles/edit", s.handleEditSubtitles)
	mux.HandleFunc("/api/timeline", s.handleGetTimeline)
	mux.HandleFunc("/api/timeline/thumbnail", s.handleServeTimelineThumbnail)
	mux.HandleFunc("/api/preview.png", s.handleServePreviewImage)
	mux.HandleFunc("/api/raw-frame.png", s.handleServeRawFrameImage)
	mux.HandleFunc("/api/overlay-image", s.handleServeOverlayImage)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("Layout Editor server listening on http://localhost:%d\n", port)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v\n", err)
		}
	}()

	return nil
}

func (s *Server) Stop() {
	if s.httpServer != nil {
		log.Println("Shutting down Layout Editor server...")
		s.httpServer.Close()
	}
}

func (s *Server) GetLayers() []actions.RenderLayer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.layers
}

func (s *Server) generatePreviewFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure output directory for preview path exists
	dir := filepath.Dir(s.previewPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Resolve input indices and overlay layers dynamically
	inputArgs, activeLayers := actions.BuildRenderingGraph(s.layers, s.inputVideo)

	// Prepend -y and -ss <s.currentTime> to inputArgs
	previewArgs := []string{"-y", "-ss", fmt.Sprintf("%.3f", s.currentTime)}
	previewArgs = append(previewArgs, inputArgs...)

	// Only burn subtitles if the ASS file actually exists on disk
	activeASSPath := ""
	if s.assPath != "" {
		if _, err := os.Stat(s.assPath); err == nil {
			activeASSPath = s.assPath
		}
	}

	return s.previewService.GeneratePreview(previewArgs, activeLayers, s.previewPath, activeASSPath)
}

func (s *Server) handleGetLayout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var layersInfo []LayerResponse
	for _, l := range s.layers {
		cropX, cropY, cropW, cropH := l.GetCrop()
		
		inputW, inputH := 0, 0
		for _, f := range l.Filters {
			if cf, ok := f.(ffmpeg.CropPercent); ok {
				inputW = cf.InputWidth
				inputH = cf.InputHeight
				break
			}
		}
		if inputW == 0 || inputH == 0 {
			if l.ImagePath != "" {
				inputW, inputH = actions.GetImageDimensions(l.ImagePath)
			}
		}
		if inputW == 0 {
			inputW = 1920
		}
		if inputH == 0 {
			inputH = 1080
		}

		layersInfo = append(layersInfo, LayerResponse{
			Name:              l.Name,
			X:                 l.X,
			Y:                 l.Y,
			Width:             l.GetWidth(),
			Height:            l.GetHeight(),
			Visible:           l.Visible,
			ZIndex:            l.ZIndex,
			ScaleMode:         l.GetScaleMode(),
			CropXPercent:      cropX,
			CropYPercent:      cropY,
			CropWidthPercent:  cropW,
			CropHeightPercent: cropH,
			InputWidth:        inputW,
			InputHeight:       inputH,
		})
	}

	resp := LayoutResponse{
		Width:  s.preset.Crop.Width,
		Height: s.preset.Crop.Height,
		Layers: layersInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleUpdatePosition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i, l := range s.layers {
		if l.Name == req.Name {
			s.layers[i].X = req.X
			s.layers[i].Y = req.Y
			s.layers[i].Visible = req.Visible
			s.layers[i].ZIndex = req.ZIndex
			s.layers[i].SetCrop(req.CropXPercent, req.CropYPercent, req.CropWidthPercent, req.CropHeightPercent)
			found = true
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if found {
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "layer not found"})
	}
}

func (s *Server) handleUpdateSize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i, l := range s.layers {
		if l.Name == req.Name {
			s.layers[i].SetSize(req.Width, req.Height)
			s.layers[i].SetScaleMode(req.ScaleMode)
			found = true
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if found {
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "layer not found"})
	}
}

func (s *Server) handleRegeneratePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	timeStr := r.URL.Query().Get("time")
	if timeStr != "" {
		var t float64
		if _, err := fmt.Sscanf(timeStr, "%f", &t); err == nil {
			s.mu.Lock()
			s.currentTime = t
			s.mu.Unlock()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := s.generatePreviewFile(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleSaveLayout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update the underlying preset struct from layers
	for _, l := range s.layers {
		// Visible boolean pointer assignment helper
		visibleVal := l.Visible
		var visiblePtr *bool = &visibleVal
		cropX, cropY, cropW, cropH := l.GetCrop()

		switch l.Name {
		case "game":
			s.preset.Gameplay.X = l.X
			s.preset.Gameplay.Y = l.Y
			s.preset.Gameplay.Width = l.GetWidth()
			s.preset.Gameplay.Height = l.GetHeight()
			s.preset.Gameplay.Visible = visiblePtr
			s.preset.Gameplay.ZIndex = l.ZIndex
			s.preset.Gameplay.CropXPercent = cropX
			s.preset.Gameplay.CropYPercent = cropY
			s.preset.Gameplay.CropWidthPercent = cropW
			s.preset.Gameplay.CropHeightPercent = cropH
			s.preset.Gameplay.ScaleMode = l.GetScaleMode()
		case "cam":
			s.preset.Facecam.X = l.X
			s.preset.Facecam.Y = l.Y
			s.preset.Facecam.Width = l.GetWidth()
			s.preset.Facecam.Height = l.GetHeight()
			s.preset.Facecam.Visible = visiblePtr
			s.preset.Facecam.ZIndex = l.ZIndex
			s.preset.Facecam.CropXPercent = cropX
			s.preset.Facecam.CropYPercent = cropY
			s.preset.Facecam.CropWidthPercent = cropW
			s.preset.Facecam.CropHeightPercent = cropH
			s.preset.Facecam.ScaleMode = l.GetScaleMode()
		case "overlay":
			s.preset.Overlay.X = l.X
			s.preset.Overlay.Y = l.Y
			s.preset.Overlay.Width = l.GetWidth()
			s.preset.Overlay.Height = l.GetHeight()
			s.preset.Overlay.Visible = visiblePtr
			s.preset.Overlay.ZIndex = l.ZIndex
			s.preset.Overlay.CropXPercent = cropX
			s.preset.Overlay.CropYPercent = cropY
			s.preset.Overlay.CropWidthPercent = cropW
			s.preset.Overlay.CropHeightPercent = cropH
			s.preset.Overlay.ScaleMode = l.GetScaleMode()
		}
	}

	// Marshal and write back to disk
	data, err := json.MarshalIndent(s.preset, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	if err := os.WriteFile(s.presetPath, data, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleServePreviewImage(w http.ResponseWriter, r *http.Request) {
	// Serve the output PNG preview file directly from the filesystem
	http.ServeFile(w, r, s.previewPath)
}

func (s *Server) generateRawFrameFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rawFramePath := "output/raw_frame.png"
	dir := filepath.Dir(rawFramePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Extract the first frame using FFmpeg without filters.
	args := []string{
		"-y",
		"-ss", "00:00:00",
		"-i", s.inputVideo,
		"-vframes", "1",
		rawFramePath,
	}

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg raw frame extraction failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (s *Server) handleServeRawFrameImage(w http.ResponseWriter, r *http.Request) {
	timeStr := r.URL.Query().Get("time")
	if timeStr == "" {
		http.ServeFile(w, r, "output/raw_frame.png")
		return
	}

	var t float64
	if _, err := fmt.Sscanf(timeStr, "%f", &t); err != nil {
		http.ServeFile(w, r, "output/raw_frame.png")
		return
	}

	if t < 0 {
		t = 0
	}

	// Generate safe filename for cache
	safeTimeStr := strings.ReplaceAll(fmt.Sprintf("%.3f", t), ".", "_")
	cachedFramePath := filepath.Join("output", "frames", fmt.Sprintf("frame_%s.png", safeTimeStr))

	// Check if already cached
	if _, err := os.Stat(cachedFramePath); err == nil {
		http.ServeFile(w, r, cachedFramePath)
		return
	}

	// Extract frame at that timestamp
	if err := os.MkdirAll(filepath.Dir(cachedFramePath), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", t),
		"-i", s.inputVideo,
		"-vframes", "1",
		cachedFramePath,
	}

	cmd := exec.Command("ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("FFmpeg dynamic frame extraction failed for time %.3f: %v, output: %s", t, err, string(output))
		http.ServeFile(w, r, "output/raw_frame.png")
		return
	}

	http.ServeFile(w, r, cachedFramePath)
}

func (s *Server) handleServeOverlayImage(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	imgPath := s.preset.Overlay.Image
	s.mu.Unlock()

	if imgPath != "" {
		http.ServeFile(w, r, imgPath)
	} else {
		http.Error(w, "No overlay image configured", http.StatusNotFound)
	}
}

func (s *Server) handleExportLayout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "Name cannot be empty", http.StatusBadRequest)
		return
	}

	// Clean name to prevent directory traversal
	name = filepath.Base(name)
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure layouts directory exists
	layoutsDir := "layouts"
	if err := os.MkdirAll(layoutsDir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	destPath := filepath.Join(layoutsDir, name)

	// Sync preset from layers in-memory state before exporting
	for _, l := range s.layers {
		visibleVal := l.Visible
		var visiblePtr *bool = &visibleVal
		cropX, cropY, cropW, cropH := l.GetCrop()

		switch l.Name {
		case "game":
			s.preset.Gameplay.X = l.X
			s.preset.Gameplay.Y = l.Y
			s.preset.Gameplay.Width = l.GetWidth()
			s.preset.Gameplay.Height = l.GetHeight()
			s.preset.Gameplay.Visible = visiblePtr
			s.preset.Gameplay.ZIndex = l.ZIndex
			s.preset.Gameplay.CropXPercent = cropX
			s.preset.Gameplay.CropYPercent = cropY
			s.preset.Gameplay.CropWidthPercent = cropW
			s.preset.Gameplay.CropHeightPercent = cropH
			s.preset.Gameplay.ScaleMode = l.GetScaleMode()
		case "cam":
			s.preset.Facecam.X = l.X
			s.preset.Facecam.Y = l.Y
			s.preset.Facecam.Width = l.GetWidth()
			s.preset.Facecam.Height = l.GetHeight()
			s.preset.Facecam.Visible = visiblePtr
			s.preset.Facecam.ZIndex = l.ZIndex
			s.preset.Facecam.CropXPercent = cropX
			s.preset.Facecam.CropYPercent = cropY
			s.preset.Facecam.CropWidthPercent = cropW
			s.preset.Facecam.CropHeightPercent = cropH
			s.preset.Facecam.ScaleMode = l.GetScaleMode()
		case "overlay":
			s.preset.Overlay.X = l.X
			s.preset.Overlay.Y = l.Y
			s.preset.Overlay.Width = l.GetWidth()
			s.preset.Overlay.Height = l.GetHeight()
			s.preset.Overlay.Visible = visiblePtr
			s.preset.Overlay.ZIndex = l.ZIndex
			s.preset.Overlay.CropXPercent = cropX
			s.preset.Overlay.CropYPercent = cropY
			s.preset.Overlay.CropWidthPercent = cropW
			s.preset.Overlay.CropHeightPercent = cropH
			s.preset.Overlay.ScaleMode = l.GetScaleMode()
		}
	}

	data, err := json.MarshalIndent(s.preset, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

type SubtitleItemResponse struct {
	Index int     `json:"index"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type SubtitleStyleResponse struct {
	FontName     string `json:"font_name"`
	FontSize     int    `json:"font_size"`
	PrimaryColor string `json:"primary_color"`
	OutlineColor string `json:"outline_color"`
	MarginV      int    `json:"margin_v"`
	Alignment    int    `json:"alignment"`
}

type SubtitlesResponse struct {
	SRTPath   string                 `json:"srt_path"`
	Style     SubtitleStyleResponse  `json:"style"`
	Subtitles []SubtitleItemResponse `json:"subtitles"`
}

type UpdateSubtitleStyleRequest struct {
	FontName     string `json:"font_name"`
	FontSize     int    `json:"font_size"`
	PrimaryColor string `json:"primary_color"`
	OutlineColor string `json:"outline_color"`
	MarginV      int    `json:"margin_v"`
	Alignment    int    `json:"alignment"`
}

type EditSubtitlesRequest struct {
	Subtitles []SubtitleItemResponse `json:"subtitles"`
}

func (s *Server) handleGetSubtitles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var items []SubtitleItemResponse
	if s.srtPath != "" {
		if _, err := os.Stat(s.srtPath); err == nil {
			track, err := subtitle.ParseSRT(s.srtPath)
			if err == nil && track != nil {
				for _, it := range track.Items {
					items = append(items, SubtitleItemResponse{
						Index: it.Index,
						Start: it.Start,
						End:   it.End,
						Text:  it.Text,
					})
				}
			}
		}
	}

	// Current style
	pStyle := getSubtitleStyle(s.preset)
	styleResp := SubtitleStyleResponse{
		FontName:     pStyle.FontName,
		FontSize:     pStyle.FontSize,
		PrimaryColor: fmt.Sprintf("#%02X%02X%02X", pStyle.PrimaryColor.R, pStyle.PrimaryColor.G, pStyle.PrimaryColor.B),
		OutlineColor: fmt.Sprintf("#%02X%02X%02X", pStyle.OutlineColor.R, pStyle.OutlineColor.G, pStyle.OutlineColor.B),
		MarginV:      pStyle.MarginV,
		Alignment:    pStyle.Alignment,
	}

	resp := SubtitlesResponse{
		SRTPath:   s.srtPath,
		Style:     styleResp,
		Subtitles: items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleUpdateSubtitlesStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateSubtitleStyleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.preset.Subtitles == nil {
		s.preset.Subtitles = &preset.SubtitlePresetConfig{}
	}

	s.preset.Subtitles.FontName = req.FontName
	s.preset.Subtitles.FontSize = req.FontSize
	s.preset.Subtitles.PrimaryColor = req.PrimaryColor
	s.preset.Subtitles.OutlineColor = req.OutlineColor
	s.preset.Subtitles.MarginV = req.MarginV
	s.preset.Subtitles.Alignment = req.Alignment

	// Save preset to disk
	data, err := json.MarshalIndent(s.preset, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	if err := os.WriteFile(s.presetPath, data, 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	// Regenerate ASS file
	if err := s.regenerateASSFile(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) handleEditSubtitles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EditSubtitlesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srtPath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "srt path not configured"})
		return
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(s.srtPath), 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	track := subtitle.NewSubtitleTrack()
	for _, item := range req.Subtitles {
		track.Add(subtitle.Subtitle{
			Index: item.Index,
			Start: item.Start,
			End:   item.End,
			Text:  item.Text,
		})
	}

	if err := track.SaveSRT(s.srtPath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	// Regenerate ASS file
	if err := s.regenerateASSFile(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// regenerateASSFile helper (must be called with s.mu locked)
func (s *Server) regenerateASSFile() error {
	if s.srtPath == "" || s.assPath == "" {
		return nil
	}

	// Check if SRT exists; if not, there is nothing to generate
	if _, err := os.Stat(s.srtPath); os.IsNotExist(err) {
		return nil
	}

	track, err := subtitle.ParseSRT(s.srtPath)
	if err != nil {
		return fmt.Errorf("failed to parse SRT for ASS generation: %w", err)
	}

	style := getSubtitleStyle(s.preset)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.assPath), 0755); err != nil {
		return err
	}

	if err := subtitle.GenerateASSFromTrack(track, style, s.assPath); err != nil {
		return fmt.Errorf("failed to generate ASS: %w", err)
	}

	return nil
}

func getSubtitleStyle(p *preset.Preset) subtitle.SubtitleStyle {
	style := subtitle.TikTokStyle()
	if p.Subtitles == nil {
		return style
	}
	if p.Subtitles.FontName != "" {
		style.FontName = p.Subtitles.FontName
	}
	if p.Subtitles.FontSize != 0 {
		style.FontSize = p.Subtitles.FontSize
	}
	if p.Subtitles.PrimaryColor != "" {
		if c, err := parseHexColor(p.Subtitles.PrimaryColor); err == nil {
			style.PrimaryColor = c
		}
	}
	if p.Subtitles.OutlineColor != "" {
		if c, err := parseHexColor(p.Subtitles.OutlineColor); err == nil {
			style.OutlineColor = c
		}
	}
	if p.Subtitles.MarginV != 0 {
		style.MarginV = p.Subtitles.MarginV
	}
	if p.Subtitles.Alignment != 0 {
		style.Alignment = p.Subtitles.Alignment
	}
	return style
}

func parseHexColor(hex string) (subtitle.Color, error) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return subtitle.Color{}, fmt.Errorf("invalid hex color: %s", hex)
	}
	var r, g, b int
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return subtitle.Color{}, err
	}
	return subtitle.NewColor(r, g, b), nil
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

type TimelineResponse struct {
	Duration float64 `json:"duration"`
	Count    int     `json:"count"`
}

func (s *Server) handleGetTimeline(w http.ResponseWriter, r *http.Request) {
	duration, err := getVideoDuration(s.inputVideo)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(TimelineResponse{
		Duration: duration,
		Count:    12, // 12 thumbnails on the timeline
	})
}

func (s *Server) handleServeTimelineThumbnail(w http.ResponseWriter, r *http.Request) {
	indexStr := r.URL.Query().Get("index")
	var idx int
	if _, err := fmt.Sscanf(indexStr, "%d", &idx); err != nil {
		http.Error(w, "invalid index parameter", http.StatusBadRequest)
		return
	}

	duration, err := getVideoDuration(s.inputVideo)
	if err != nil {
		http.Error(w, "failed to get video duration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	count := 12
	if idx < 0 || idx >= count {
		http.Error(w, "index out of bounds", http.StatusBadRequest)
		return
	}

	// Calculate target timestamp
	t := float64(idx) * (duration / float64(count))

	// Ensure thumb directory exists
	thumbPath := filepath.Join("output", "thumbnails", fmt.Sprintf("thumb_%d.png", idx))
	if err := os.MkdirAll(filepath.Dir(thumbPath), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If already cached, serve it
	if _, err := os.Stat(thumbPath); err == nil {
		http.ServeFile(w, r, thumbPath)
		return
	}

	// Extract and resize thumbnail to width 160px using fast seek
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", t),
		"-i", s.inputVideo,
		"-vframes", "1",
		"-vf", "scale=160:-1",
		thumbPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("FFmpeg thumbnail extraction failed for index %d: %v, output: %s", idx, err, string(output))
		http.Error(w, "FFmpeg failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, thumbPath)
}
