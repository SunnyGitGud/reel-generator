package renderer

import (
	"fmt"
	"os"
	"os/exec"
)

func Render(
	input string,
	output string,
	filterGraph string,
) error {

	args := []string{
		"-i", input,
		"-filter_complex", filterGraph,
		"-c:a", "aac",
		"-c:v", "libx264",
		"-preset", "fast",
		"-y",
		output,
	}

	fmt.Println("ffmpeg", args)

	cmd := exec.Command("ffmpeg", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
