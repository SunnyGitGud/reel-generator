package subtitle

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParseSRT parses an SRT subtitle file and returns a SubtitleTrack
func ParseSRT(path string) (*SubtitleTrack, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read SRT file: %w", err)
	}

	// Parse content
	return parseSRTContent(string(content))
}

// ParseSRTContent parses SRT content from a string
func ParseSRTContent(content string) (*SubtitleTrack, error) {
	return parseSRTContent(content)
}

// parseSRTContent is the internal parser
func parseSRTContent(content string) (*SubtitleTrack, error) {
	track := NewSubtitleTrack()

	// Split by blank lines to get subtitle blocks
	blocks := strings.Split(content, "\n\n")

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		sub, err := parseSRTBlock(block)
		if err != nil {
			return nil, err
		}

		if sub != nil {
			track.Add(*sub)
		}
	}

	return track, nil
}

// parseSRTBlock parses a single SRT block
// Example:
// 1
// 00:00:01,000 --> 00:00:03,000
// omgz captions subtitles ezzz
func parseSRTBlock(block string) (*Subtitle, error) {
	lines := strings.Split(strings.TrimSpace(block), "\n")

	if len(lines) < 3 {
		return nil, nil // Skip invalid blocks
	}

	// Line 1: Index (skip it)
	// Line 2: Timestamps
	timeLine := strings.TrimSpace(lines[1])

	// Parse timestamps
	start, end, err := parseTimestamps(timeLine)
	if err != nil {
		return nil, err
	}

	// Lines 3+: Text (join with spaces for multiline subtitles)
	text := strings.TrimSpace(strings.Join(lines[2:], " "))

	return &Subtitle{
		Start: start,
		End:   end,
		Text:  text,
	}, nil
}

// parseTimestamps parses SRT timestamp format: "00:00:01,000 --> 00:00:03,000"
func parseTimestamps(timeLine string) (float64, float64, error) {
	// Regex to match SRT timestamp format
	pattern := `(\d{1,2}):(\d{2}):(\d{2}),(\d{3})\s*-->\s*(\d{1,2}):(\d{2}):(\d{2}),(\d{3})`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(timeLine)
	if len(matches) != 9 {
		return 0, 0, fmt.Errorf("invalid timestamp format: %s", timeLine)
	}

	// Parse start time
	startHours := parseIntOrZero(matches[1])
	startMinutes := parseIntOrZero(matches[2])
	startSeconds := parseIntOrZero(matches[3])
	startMillis := parseIntOrZero(matches[4])

	startTime := float64(startHours*3600+startMinutes*60+startSeconds) + float64(startMillis)/1000.0

	// Parse end time
	endHours := parseIntOrZero(matches[5])
	endMinutes := parseIntOrZero(matches[6])
	endSeconds := parseIntOrZero(matches[7])
	endMillis := parseIntOrZero(matches[8])

	endTime := float64(endHours*3600+endMinutes*60+endSeconds) + float64(endMillis)/1000.0

	return startTime, endTime, nil
}

// parseIntOrZero parses an int from string, returns 0 if invalid
func parseIntOrZero(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// SaveSRT saves the subtitle track back to an SRT file
func (st *SubtitleTrack) SaveSRT(path string) error {
	var sb strings.Builder
	for i, item := range st.Items {
		sb.WriteString(fmt.Sprintf("%d\n", i+1))
		sb.WriteString(fmt.Sprintf("%s --> %s\n", formatSRTTimestamp(item.Start), formatSRTTimestamp(item.End)))
		sb.WriteString(fmt.Sprintf("%s\n\n", item.Text))
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func formatSRTTimestamp(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	millis := int((seconds - float64(int(seconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}
