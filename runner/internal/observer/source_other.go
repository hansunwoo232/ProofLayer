//go:build !windows

package observer

import (
	"context"
	"errors"
	"time"
)

type WindowsEventSource struct{}

func NewWindowsEventSource() *WindowsEventSource {
	return &WindowsEventSource{}
}

func (*WindowsEventSource) RecentProcessEvents(context.Context, time.Time) ([]Event, error) {
	return nil, errors.New("Windows Event Log source is unavailable on this platform")
}
