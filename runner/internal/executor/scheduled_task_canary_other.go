//go:build !windows

package executor

import "context"

func runScheduledTaskCanary(context.Context, string) error {
	return ErrUnsupportedPlatform
}

func cleanupScheduledTaskCanary(ctx context.Context, _ string) error {
	return ctx.Err()
}

func verifyScheduledTaskCanaryAbsent(ctx context.Context, _ string) error {
	return ctx.Err()
}
