//go:build windows

package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const scheduledTasksExecutable = `C:\Windows\System32\schtasks.exe`

func runScheduledTaskCanary(ctx context.Context, taskName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return runScheduledTasksCommand(ctx, scheduledTaskCreateArguments(taskName))
}

func cleanupScheduledTaskCanary(ctx context.Context, taskName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := os.Stat(scheduledTaskArtifactPath(taskName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect scheduled task before cleanup: %w", err)
	}
	return runScheduledTasksCommand(ctx, scheduledTaskDeleteArguments(taskName))
}

func verifyScheduledTaskCanaryAbsent(ctx context.Context, taskName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := os.Stat(scheduledTaskArtifactPath(taskName))
	if errors.Is(err, os.ErrNotExist) {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("verify scheduled task absence: %w", err)
	}
	return ErrArtifactPresent
}

func scheduledTaskArtifactPath(taskName string) string {
	windowsRoot := os.Getenv("SystemRoot")
	if windowsRoot == "" {
		windowsRoot = `C:\Windows`
	}
	return filepath.Join(windowsRoot, "System32", "Tasks", taskName)
}

func runScheduledTasksCommand(ctx context.Context, arguments []string) error {
	command := exec.CommandContext(ctx, scheduledTasksExecutable, arguments...)
	command.Stdin = nil
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Run(); err != nil {
		return fmt.Errorf("run fixed scheduled task operation: %w", err)
	}
	return ctx.Err()
}
