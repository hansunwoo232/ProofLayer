package executor

import (
	"context"
	"errors"
	"strings"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
)

const (
	scheduledTaskCanaryPrefix = "ProofLayer_"
	scheduledTaskCanaryAction = `C:\Windows\System32\cmd.exe /d /c exit 0`
)

func RunScheduledTaskCanary(ctx context.Context, correlationID string) error {
	taskName, err := scheduledTaskCanaryName(correlationID)
	if err != nil {
		return err
	}
	return runScheduledTaskCanary(ctx, taskName)
}

func CleanupScheduledTaskCanary(ctx context.Context, correlationID string) error {
	taskName, err := scheduledTaskCanaryName(correlationID)
	if err != nil {
		return err
	}
	return cleanupScheduledTaskCanary(ctx, taskName)
}

func VerifyScheduledTaskCanaryAbsent(ctx context.Context, correlationID string) error {
	taskName, err := scheduledTaskCanaryName(correlationID)
	if err != nil {
		return err
	}
	return verifyScheduledTaskCanaryAbsent(ctx, taskName)
}

func scheduledTaskCanaryName(correlationID string) (string, error) {
	if !correlation.Valid(correlationID) {
		return "", errors.New("invalid correlation ID")
	}
	return scheduledTaskCanaryPrefix + strings.TrimPrefix(correlationID, "PL-"), nil
}

func scheduledTaskCreateArguments(taskName string) []string {
	return []string{
		"/Create",
		"/TN", taskName,
		"/TR", scheduledTaskCanaryAction,
		"/SC", "ONLOGON",
		"/RL", "LIMITED",
		"/F",
	}
}

func scheduledTaskDeleteArguments(taskName string) []string {
	return []string{"/Delete", "/TN", taskName, "/F"}
}

func scheduledTaskCanaryHandler() ScenarioHandler {
	return handlerFunctions{
		execute:      RunScheduledTaskCanary,
		cleanup:      CleanupScheduledTaskCanary,
		verifyAbsent: VerifyScheduledTaskCanaryAbsent,
	}
}
