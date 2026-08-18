//go:build windows

package observer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

const (
	sysmonChannel   = "Microsoft-Windows-Sysmon/Operational"
	maximumXMLBytes = 512 * 1024
)

type WindowsEventSource struct{}

func NewWindowsEventSource() *WindowsEventSource {
	return &WindowsEventSource{}
}

func (*WindowsEventSource) RecentProcessEvents(ctx context.Context, since time.Time) ([]Event, error) {
	lookback := time.Since(since).Round(time.Millisecond) + 2*time.Second
	// Keep the query wider than the evidence acceptance window. Sysmon can
	// publish shortly after the process exits, while the observer still rejects
	// events older than execution start minus two seconds.
	if lookback < 10*time.Second {
		lookback = 10 * time.Second
	}
	if lookback > time.Minute {
		lookback = time.Minute
	}
	query := "*[System[(EventID=1) and TimeCreated[timediff(@SystemTime) <= " +
		strconv.FormatInt(lookback.Milliseconds(), 10) + "]]]"

	command := exec.CommandContext(
		ctx,
		`C:\Windows\System32\wevtutil.exe`,
		"qe",
		sysmonChannel,
		"/q:"+query,
		"/rd:true",
		"/c:50",
		"/f:xml",
		"/uni:true",
	)
	var output cappedBuffer
	output.maximum = maximumXMLBytes
	var stderr cappedBuffer
	stderr.maximum = 4096
	command.Stdout = &output
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(output.err, ErrEvidenceLimit) {
			return nil, ErrEvidenceLimit
		}
		return nil, fmt.Errorf("%w: command", ErrWindowsEventQuery)
	}

	events, err := parseWindowsEventXML(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWindowsEventXML, err)
	}
	return events, nil
}

type cappedBuffer struct {
	buffer  bytes.Buffer
	maximum int
	err     error
}

func (buffer *cappedBuffer) Write(value []byte) (int, error) {
	if buffer.buffer.Len()+len(value) > buffer.maximum {
		buffer.err = ErrEvidenceLimit
		return 0, buffer.err
	}
	return buffer.buffer.Write(value)
}

func (buffer *cappedBuffer) String() string {
	return buffer.buffer.String()
}

func (buffer *cappedBuffer) Bytes() []byte {
	return buffer.buffer.Bytes()
}
