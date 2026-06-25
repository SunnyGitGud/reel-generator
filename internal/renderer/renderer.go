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
			"-b:a", "384k",
			"-ar", "48000",
			"-c:v", "h264_nvenc",
			"-preset", "slow",
			"-rc:v", "vbr",
			"-cq", "19",
			"-b:v", "0",
			"-pix_fmt", "yuv420p",
			"-profile:v", "high",
			"-y",
			output,
		)
	} else {
		args = append(args,
			"-filter_complex", filterGraph,
			"-c:a", "aac",
			"-b:a", "384k",
			"-ar", "48000",
			"-c:v", "libx264",
			"-preset", "slow",
			"-crf", "18",
			"-pix_fmt", "yuv420p",
			"-profile:v", "high",
			"-level:v", "4.1",
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
