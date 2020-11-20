package modules

import (
	"fmt"
	"os/exec"
)

const ffmpegCmd = "ffmpeg"

func ffmpegEnabled() error {
	_, pErr := exec.LookPath(ffmpegCmd)
	if pErr != nil {
		return fmt.Errorf("external command does not exist: %s", ffmpegCmd)
	}
	return nil
}
