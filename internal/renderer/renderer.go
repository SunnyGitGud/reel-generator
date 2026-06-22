package renderer

import (
	"fmt"
	"os"
	"os/exec"
)

func Render(
	inputArgs []string,
	output string,
	filterGraph string,
	nvenc bool,
) error {

	args := []string{}
	args = append(args, inputArgs...)
	
	if nvenc {
		args = append(args,
			"-filter_complex", filterGraph,
			"-c:a", "aac",
			"-c:v", "h264_nvenc",
			"-y",
			output,
		)
	} else {
		args = append(args,
			"-filter_complex", filterGraph,
			"-c:a", "aac",
			"-c:v", "libx264",
			"-preset", "fast",
			"-y",
			output,
		)
	}

	fmt.Println("ffmpeg", args)

	cmd := exec.Command("ffmpeg", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
