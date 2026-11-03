package reliable

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"

	"github.com/kandev/kandev/internal/common/winproc"
	"golang.org/x/sys/windows"
)

func prepareOwnedCommand(cmd *exec.Cmd) {
	attributes := syscall.SysProcAttr{}
	if cmd.SysProcAttr != nil {
		attributes = *cmd.SysProcAttr
	}
	attributes.CreationFlags |= windows.CREATE_SUSPENDED
	cmd.SysProcAttr = &attributes
}

func ownedProcessTreeAttested() bool { return true }

func isolateProcess(cmd *exec.Cmd) (func() error, error) {
	job, err := winproc.InstallKillOnCloseJobForSuspendedCommand(cmd)
	if err != nil {
		return nil, err
	}
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return errors.Join(job.TerminateAndWait(ctx), job.Close())
	}, nil
}
