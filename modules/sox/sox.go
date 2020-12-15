package sox

import (
	"fmt"
	"os/exec"
)

var SoxCmd = "sox"

func soxEnabled() error {
	_, pErr := exec.LookPath(SoxCmd)
	if pErr != nil {
		return fmt.Errorf("external command does not exist: %s", SoxCmd)
	}
	return nil
}
