//go:build !windows

package reliable

import (
	"errors"
	"os"
	"os/exec"
)

func prepareOwnedCommand(*exec.Cmd) {}

func ownedProcessTreeAttested() bool { return false }

func isolateProcess(cmd *exec.Cmd) (func() error, error) {
	return func() error {
		err := cmd.Process.Kill()
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}, nil
}
