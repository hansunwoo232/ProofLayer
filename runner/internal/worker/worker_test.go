package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/controlplane"
	"github.com/hansunwoo232/ProofLayer/runner/internal/executor"
	"github.com/hansunwoo232/ProofLayer/runner/internal/job"
	"github.com/hansunwoo232/ProofLayer/runner/internal/observer"
	"github.com/hansunwoo232/ProofLayer/runner/internal/splunk"
)

const testCorrelationID = "PL-0123456789ABCDEF0123456789ABCDEF"

type fakeControlPlane struct {
	job      controlplane.Job
	updates  []controlplane.StageUpdate
	acked    bool
	complete bool
}

func (fake *fakeControlPlane) Lease(context.Context) (controlplane.Job, bool, error) {
	return fake.job, true, nil
}
func (fake *fakeControlPlane) Acknowledge(_ context.Context, _ string, accepted bool) error {
	fake.acked = accepted
	return nil
}
func (fake *fakeControlPlane) UpdateStage(_ context.Context, _ string, update controlplane.StageUpdate) error {
	fake.updates = append(fake.updates, update)
	return nil
}
func (fake *fakeControlPlane) Complete(context.Context, string) error {
	fake.complete = true
	return nil
}

type fakeExecutor struct{ result executor.Result }

func (fake fakeExecutor) Execute(context.Context, job.ExecutionRequest) executor.Result {
	return fake.result
}

type fakeEndpoint struct {
	evidence observer.Evidence
	err      error
}

func (fake fakeEndpoint) Observe(context.Context, string, time.Time) (observer.Evidence, error) {
	return fake.evidence, fake.err
}

type fakeExporter struct{ err error }

func (fake fakeExporter) Export(context.Context, string, observer.Evidence) error { return fake.err }

type fakeSIEM struct {
	evidence     splunk.CorrelationEvidence
	searchErr    error
	detection    splunk.DetectionEvidence
	detectionErr error
}

func (fake fakeSIEM) SearchExact(context.Context, string, splunk.SearchWindow) (splunk.CorrelationEvidence, error) {
	return fake.evidence, fake.searchErr
}
func (fake fakeSIEM) SearchDetection(context.Context, string, splunk.SearchWindow, splunk.DetectionPlan) (splunk.DetectionEvidence, error) {
	return fake.detection, fake.detectionErr
}

func baseDependencies() (*fakeControlPlane, fakeExecutor, fakeEndpoint, *fakeExporter, *fakeSIEM) {
	now := time.Now().UTC()
	cp := &fakeControlPlane{job: controlplane.Job{
		JobID: "7ba7b811-9dad-41d1-80b4-00c04fd430c8", CorrelationID: testCorrelationID,
		EnvironmentID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
		HostID:        "6ba7b811-9dad-41d1-80b4-00c04fd430c8",
		ScenarioID:    "windows-process-marker", ScenarioVersion: "0.1.0", Parameters: map[string]any{},
	}}
	exec := fakeExecutor{result: executor.Result{
		Status: executor.StatusPassed, CleanupStatus: executor.StatusPassed,
		StartedAt: now.Add(-time.Second), CompletedAt: now, LatencyMS: 10,
	}}
	endpoint := fakeEndpoint{evidence: observer.Evidence{
		Provider: "Microsoft-Windows-Sysmon", EventID: 1, RecordID: 7,
		TimeCreatedUTC: now.Add(-time.Second), ObservedAtUTC: now,
	}}
	exporter := fakeExporter{}
	siem := fakeSIEM{
		evidence: splunk.CorrelationEvidence{
			CorrelationID: testCorrelationID, Provider: "Microsoft-Windows-Sysmon", EventID: 1,
			RecordID: 7, EndpointEventTime: now.Add(-time.Second), IngestionLatencyMS: 25,
			FieldPresence: map[string]bool{"host.name": true, "process.name": true, "process.command_line": true, "user.name": true},
		},
		detection: splunk.DetectionEvidence{Status: splunk.DetectionStatusPassed, Detected: true},
	}
	return cp, exec, endpoint, &exporter, &siem
}

func TestRunOncePublishesCompletePipeline(t *testing.T) {
	cp, exec, endpoint, exporter, siem := baseDependencies()
	value, _ := New(cp, exec, endpoint, exporter, siem)
	value.polling = splunk.PollingPolicy{Timeout: time.Second, Interval: time.Millisecond * 250, MaximumAttempts: 1}
	result, err := value.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || !cp.acked || !cp.complete {
		t.Fatalf("result = %+v, acked = %v, complete = %v", result, cp.acked, cp.complete)
	}
	want := []string{
		"execution:running", "execution:passed", "endpoint_telemetry:running", "endpoint_telemetry:passed",
		"siem_ingestion:running", "siem_ingestion:passed", "field_validation:running", "field_validation:passed",
		"detection:running", "detection:passed", "alert:not_tested", "cleanup:passed",
	}
	if got := updateSequence(cp.updates); !equalStrings(got, want) {
		t.Fatalf("updates = %v, want %v", got, want)
	}
}

func TestRunOnceStopsAfterMissingEndpointAndStillCompletesCleanup(t *testing.T) {
	cp, exec, endpoint, exporter, siem := baseDependencies()
	endpoint.err = observer.ErrEventNotFound
	value, _ := New(cp, exec, endpoint, exporter, siem)
	result, err := value.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" || result.ErrorCode != "endpoint_event_missing" || !cp.complete {
		t.Fatalf("result = %+v, complete = %v", result, cp.complete)
	}
	wantTail := []string{"siem_ingestion:not_tested", "field_validation:not_tested", "detection:not_tested", "alert:not_tested", "cleanup:passed"}
	got := updateSequence(cp.updates)
	if !equalStrings(got[len(got)-len(wantTail):], wantTail) {
		t.Fatalf("updates = %v", got)
	}
}

func TestRunOnceReportsFieldAndDetectionFailures(t *testing.T) {
	for name, mutate := range map[string]func(*fakeSIEM){
		"field":     func(value *fakeSIEM) { value.evidence.FieldPresence["process.command_line"] = false },
		"detection": func(value *fakeSIEM) { value.detection = splunk.DetectionEvidence{} },
	} {
		t.Run(name, func(t *testing.T) {
			cp, exec, endpoint, exporter, siem := baseDependencies()
			mutate(siem)
			value, _ := New(cp, exec, endpoint, exporter, siem)
			value.polling = splunk.PollingPolicy{Timeout: time.Second, Interval: time.Millisecond * 250, MaximumAttempts: 1}
			result, err := value.RunOnce(context.Background())
			if err != nil || result.Status != "failed" || !cp.complete {
				t.Fatalf("result = %+v, err = %v", result, err)
			}
		})
	}
}

func TestRunOnceRejectsUnsupportedScenario(t *testing.T) {
	cp, exec, endpoint, exporter, siem := baseDependencies()
	cp.job.ScenarioID = "windows-registry-run-key-canary"
	value, _ := New(cp, exec, endpoint, exporter, siem)
	result, err := value.RunOnce(context.Background())
	if err != nil || result.Status != "rejected" || cp.acked || cp.complete {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
}

func TestNewRejectsMissingDependencies(t *testing.T) {
	if _, err := New(nil, nil, nil, nil, nil); !errors.Is(err, ErrInvalidDependencies) {
		t.Fatalf("error = %v", err)
	}
}

func updateSequence(updates []controlplane.StageUpdate) []string {
	result := make([]string, 0, len(updates))
	for _, update := range updates {
		result = append(result, update.Stage+":"+update.Status)
	}
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
