//go:build windows

package executor

import (
	"context"
	"io"
	"os/exec"
	"syscall"
)

func runProcessMarker(ctx context.Context, correlationID string) error {
	command := exec.CommandContext(
		ctx,
		`C:\Windows\System32\cmd.exe`,
		"/d",
		"/s",
		"/c",
		"echo "+correlationID+" >NUL",
	)
	command.Stdin = nil
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.Run()
}
