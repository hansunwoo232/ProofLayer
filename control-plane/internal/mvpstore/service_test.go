package mvpstore

import (
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/runqueue"
)

const (
	testEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	testHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
	testRunnerID      = "6ba7b812-9dad-41d1-80b4-00c04fd430c8"
)

func testStore(t *testing.T, now *time.Time) *Service {
	t.Helper()
	service, err := New(Config{
		WorkspaceID: "8ba7b810-9dad-41d1-80b4-00c04fd430c8", EnvironmentID: testEnvironmentID,
		HostID: testHostID, RunnerID: testRunnerID, HostName: "WIN-LAB-01", RunnerVersion: "0.1.0",
		Now: func() time.Time { return *now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func testQueue(t *testing.T, now *time.Time) *runqueue.Queue {
	t.Helper()
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	queue, err := runqueue.New(runqueue.Config{
		Capacity: 4, EnvironmentID: testEnvironmentID, HostID: testHostID,
		RequestedBy: "7ba7b811-9dad-41d1-80b4-00c04fd430c8", SigningKeyID: "test-key",
		SigningPrivateKey: key, Now: func() time.Time { return *now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return queue
}

func TestScenarioCatalogAndHostAuthorization(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	service := testStore(t, &now)
	if scenarios := service.Scenarios(); len(scenarios) != 3 || !scenarios[0].CleanupRequired {
		t.Fatalf("scenarios = %+v", scenarios)
	}
	if err := service.Authorize(testHostID, "windows-process-marker", "0.1.0"); err != nil {
		t.Fatal(err)
	}
	if err := service.Authorize("00000000-0000-4000-8000-000000000000", "windows-process-marker", "0.1.0"); !errors.Is(err, ErrHostAccessDenied) {
		t.Fatalf("host error = %v", err)
	}
	if err := service.Authorize(testHostID, "arbitrary-command", "0.1.0"); !errors.Is(err, ErrScenarioInvalid) {
		t.Fatalf("scenario error = %v", err)
	}
}

func TestRunnerHeartbeatDrivesOnlineAndOfflineStatus(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	service := testStore(t, &now)
	if host := service.Hosts()[0]; host.Status != "offline" || host.LastSeenAt != nil {
		t.Fatalf("initial host = %+v", host)
	}
	if err := service.RecordRunnerSeen(testRunnerID, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if host := service.Hosts()[0]; host.Status != "online" || host.RunnerVersion != "0.2.0" || host.LastSeenAt == nil {
		t.Fatalf("online host = %+v", host)
	}
	now = now.Add(2*time.Minute + time.Second)
	if host := service.Hosts()[0]; host.Status != "offline" || host.LastSeenAt == nil {
		t.Fatalf("offline host = %+v", host)
	}
	if err := service.RecordRunnerSeen("00000000-0000-4000-8000-000000000000", "0.2.0"); !errors.Is(err, ErrHostAccessDenied) {
		t.Fatalf("wrong runner error = %v", err)
	}
}

func TestOneTimeScheduleTimezoneConflictMissedAndDispatch(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	service := testStore(t, &now)
	request := ScheduleRequest{
		SchemaVersion: SchemaVersion, HostID: testHostID,
		ScenarioID: "windows-process-marker", ScenarioVersion: "0.1.0",
		ScheduledForLocal: "2026-08-21T15:02:00", TimeZone: "Europe/Istanbul",
	}
	schedule, err := service.CreateSchedule(request)
	if err != nil {
		t.Fatal(err)
	}
	if !schedule.ScheduledForUTC.Equal(time.Date(2026, 8, 21, 12, 2, 0, 0, time.UTC)) {
		t.Fatalf("scheduled UTC = %s", schedule.ScheduledForUTC)
	}
	conflict := request
	conflict.ScheduledForLocal = "2026-08-21T15:02:30"
	if _, err := service.CreateSchedule(conflict); !errors.Is(err, ErrScheduleConflict) {
		t.Fatalf("conflict error = %v", err)
	}
	past := request
	past.ScheduledForLocal = "2026-08-21T14:59:00"
	if _, err := service.CreateSchedule(past); !errors.Is(err, ErrScheduleMissed) {
		t.Fatalf("past error = %v", err)
	}

	queue := testQueue(t, &now)
	now = time.Date(2026, 8, 21, 12, 2, 1, 0, time.UTC)
	if err := service.DispatchDue(queue); err != nil {
		t.Fatal(err)
	}
	listed := service.Schedules()
	if len(listed) != 1 || listed[0].Status != ScheduleStatusQueued || listed[0].JobID == "" || queue.Depth() != 1 {
		t.Fatalf("schedules = %+v, depth = %d", listed, queue.Depth())
	}

	missedRequest := request
	missedRequest.ScheduledForLocal = "2026-08-21T15:10:00"
	if _, err := service.CreateSchedule(missedRequest); err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 8, 21, 12, 16, 0, 0, time.UTC)
	listed = service.Schedules()
	if listed[0].Status != ScheduleStatusMissed {
		t.Fatalf("missed schedule = %+v", listed[0])
	}
}
