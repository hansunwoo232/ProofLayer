//go:build !windows

package executor

import "context"

func runRegistryCanary(context.Context, string) error {
	return ErrUnsupportedPlatform
}

func cleanupRegistryCanary(ctx context.Context, _ string) error {
	return ctx.Err()
}

func verifyRegistryCanaryAbsent(ctx context.Context, _ string) error {
	return ctx.Err()
}
