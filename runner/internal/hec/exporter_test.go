package hec

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/observer"
)

const testToken = "0123456789abcdef0123456789abcdef"

func TestExporterSendsOnlyBoundedSyntheticCanaryFields(t *testing.T) {
	var received map[string]any
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/services/collector/event" || request.Header.Get("Authorization") != "Splunk "+testToken {
			t.Fatalf("request = %s %s, authorization = %q", request.Method, request.URL.Path, request.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		return testResponse(http.StatusOK, `{"text":"Success","code":0}`), nil
	})}

	exporter, err := New(Config{
		Endpoint: "https://hec.example.invalid/services/collector/event",
		Token:    testToken, HTTPClient: client,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	correlationID := "PL-0123456789ABCDEF0123456789ABCDEF"
	if err := exporter.Export(context.Background(), correlationID, observer.Evidence{
		Provider: "Microsoft-Windows-Sysmon", EventID: 1, RecordID: 44,
		TimeCreatedUTC: now, ObservedAtUTC: now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(received)
	value := string(encoded)
	for _, expected := range []string{correlationID, "prooflayer-windows-lab", "prooflayer-synthetic-user", "cmd.exe"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("payload missing %q: %s", expected, value)
		}
	}
	for _, prohibited := range []string{"Administrator", `C:\\Users`, "_raw", "event_xml"} {
		if strings.Contains(value, prohibited) {
			t.Fatalf("payload contains prohibited value %q", prohibited)
		}
	}
}

func TestExporterRejectsInvalidMetadataAndHECResponse(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testResponse(http.StatusUnauthorized, "no"), nil
	})}
	exporter, err := New(Config{Endpoint: "https://hec.example.invalid/services/collector/event", Token: testToken, HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	if err := exporter.Export(context.Background(), "bad", observer.Evidence{}); err != ErrInvalidEvent {
		t.Fatalf("invalid event error = %v", err)
	}
	now := time.Now().UTC()
	err = exporter.Export(context.Background(), "PL-0123456789ABCDEF0123456789ABCDEF", observer.Evidence{
		Provider: "Microsoft-Windows-Sysmon", EventID: 1, RecordID: 1,
		TimeCreatedUTC: now, ObservedAtUTC: now,
	})
	if err == nil || !strings.Contains(err.Error(), "http_401") {
		t.Fatalf("rejection error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func testResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
