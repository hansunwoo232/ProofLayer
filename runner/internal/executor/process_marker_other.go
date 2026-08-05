//go:build !windows

package executor

import "context"

func runProcessMarker(context.Context, string) error {
	return ErrUnsupportedPlatform
}
