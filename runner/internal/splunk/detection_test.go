package splunk

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const detectionCorrelationID = "PL-0123456789ABCDEF0123456789ABCDEF"

func detectionWindow() SearchWindow {
	return SearchWindow{
		Earliest: time.Unix(1_754_416_800, 0).UTC(),
		Latest:   time.Unix(1_754_420_400, 0).UTC(),
	}
}

func TestInlineDetectionPresent(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		search := searchFromRequest(t, request)
		for _, required := range []string{
			"index=prooflayer_test",
			`source="prooflayer:windows-lab"`,
			"earliest=1754416800",
			"latest=1754420400",
			detectionCorrelationID,
			`detection_id="prooflayer.windows_process_marker"`,
			"head 2",
			"table correlation_id,detection_id",
		} {
			if !strings.Contains(search, required) {
				t.Errorf("inline search missing %q: %s", required, search)
			}
		}
		return response(http.StatusOK, detectionRowJSON(detectionCorrelationID)+"\n"), nil
	})}
	connector := testConnector(t, client)
	evidence, err := connector.SearchDetection(
		context.Background(),
		detectionCorrelationID,
		detectionWindow(),
		BuiltInInlineDetectionPlan(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != DetectionStatusPassed || !evidence.Detected || evidence.MatchCount != 1 {
		t.Fatalf("evidence = %+v", evidence)
	}
}

func TestDetectionAbsentIsAStableFailedResult(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, ""), nil
	})}
	connector := testConnector(t, client)
	evidence, err := connector.SearchDetection(
		context.Background(),
		detectionCorrelationID,
		detectionWindow(),
		BuiltInInlineDetectionPlan(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != DetectionStatusFailed || evidence.Detected || evidence.MatchCount != 0 {
		t.Fatalf("evidence = %+v", evidence)
	}
}

func TestDetectionAmbiguityFailsClosed(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, detectionRowJSON(detectionCorrelationID)+"\n"+detectionRowJSON(detectionCorrelationID)+"\n"), nil
	})}
	connector := testConnector(t, client)
	_, err := connector.SearchDetection(
		context.Background(),
		detectionCorrelationID,
		detectionWindow(),
		BuiltInInlineDetectionPlan(),
	)
	if !errors.Is(err, ErrAmbiguousDetection) {
		t.Fatalf("error = %v", err)
	}
}

func TestSavedSearchPlanUsesOnlyFixedNameAndParameters(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		search := searchFromRequest(t, request)
		expected := `| savedsearch "ProofLayer Windows Process Marker" correlation_id="` + detectionCorrelationID +
			`" earliest_epoch="1754416800" latest_epoch="1754420400" | head 2 | table correlation_id,detection_id`
		if search != expected {
			t.Fatalf("saved search = %q", search)
		}
		return response(http.StatusOK, detectionRowJSON(detectionCorrelationID)+"\n"), nil
	})}
	connector := testConnector(t, client)
	evidence, err := connector.SearchDetection(
		context.Background(),
		detectionCorrelationID,
		detectionWindow(),
		BuiltInSavedSearchDetectionPlan(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Mode != DetectionModeSavedSearch || evidence.RuleReference != "ProofLayer Windows Process Marker" {
		t.Fatalf("evidence = %+v", evidence)
	}
}

func TestDetectionPlansCannotBeRelaxed(t *testing.T) {
	invalid := []DetectionPlan{
		{},
		{detectionID: "custom", mode: DetectionModeInline},
		{detectionID: processMarkerDetectionID, mode: "arbitrary"},
		{detectionID: processMarkerDetectionID, mode: DetectionModeInline, savedSearchName: "Injected"},
		{detectionID: processMarkerDetectionID, mode: DetectionModeSavedSearch, savedSearchName: "Other Search"},
	}
	for index, plan := range invalid {
		if err := plan.Validate(); !errors.Is(err, ErrInvalidDetectionPlan) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func testConnector(t *testing.T, client *http.Client) *Connector {
	t.Helper()
	connector, err := New(Config{
		BaseURL:  "https://splunk.test:8089",
		Username: ObserverUsername,
		Password: testPassword,
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	return connector
}

func searchFromRequest(t *testing.T, request *http.Request) string {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatal(err)
	}
	return form.Get("search")
}

func detectionRowJSON(correlationID string) string {
	return `{"result":{"correlation_id":"` + correlationID + `","detection_id":"prooflayer.windows_process_marker"}}`
}
