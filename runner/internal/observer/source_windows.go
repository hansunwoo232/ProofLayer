//go:build windows

package observer

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	sysmonChannel   = "Microsoft-Windows-Sysmon/Operational"
	maximumXMLBytes = 512 * 1024
)

var ErrEvidenceLimit = errors.New("Windows event evidence exceeded the local limit")

type WindowsEventSource struct{}

func NewWindowsEventSource() *WindowsEventSource {
	return &WindowsEventSource{}
}

func (*WindowsEventSource) RecentProcessEvents(ctx context.Context, since time.Time) ([]Event, error) {
	lookback := time.Since(since).Round(time.Millisecond) + 2*time.Second
	if lookback < 2*time.Second {
		lookback = 2 * time.Second
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
	)
	var output cappedBuffer
	output.maximum = maximumXMLBytes
	command.Stdout = &output
	command.Stderr = &cappedBuffer{maximum: 4096}
	if err := command.Run(); err != nil {
		if errors.Is(output.err, ErrEvidenceLimit) {
			return nil, ErrEvidenceLimit
		}
		return nil, fmt.Errorf("query Sysmon channel: %w", err)
	}

	payload := strings.ReplaceAll(output.String(), `<?xml version="1.0" encoding="utf-8" standalone="yes"?>`, "")
	payload = "<Events>" + payload + "</Events>"
	var envelope xmlEnvelope
	if err := xml.Unmarshal([]byte(payload), &envelope); err != nil {
		return nil, fmt.Errorf("parse Sysmon XML: %w", err)
	}

	events := make([]Event, 0, len(envelope.Events))
	for _, parsed := range envelope.Events {
		createdAt, err := time.Parse(time.RFC3339Nano, parsed.System.TimeCreated.SystemTime)
		if err != nil {
			continue
		}
		values := make([]string, 0, len(parsed.EventData))
		for _, data := range parsed.EventData {
			values = append(values, data.Value)
		}
		events = append(events, Event{
			Provider:       parsed.System.Provider.Name,
			EventID:        parsed.System.EventID,
			RecordID:       parsed.System.RecordID,
			TimeCreatedUTC: createdAt.UTC(),
			DataValues:     values,
		})
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

type xmlEnvelope struct {
	Events []xmlEvent `xml:"Event"`
}

type xmlEvent struct {
	System struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
		EventID     int    `xml:"EventID"`
		RecordID    uint64 `xml:"EventRecordID"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
	} `xml:"System"`
	EventData []struct {
		Value string `xml:",chardata"`
	} `xml:"EventData>Data"`
}
