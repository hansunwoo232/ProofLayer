package mvpstore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hansunwoo232/ProofLayer/control-plane/internal/runqueue"
)

const (
	SchemaVersion         = "1.0"
	ScheduleStatusPlanned = "planned"
	ScheduleStatusQueued  = "queued"
	ScheduleStatusMissed  = "missed"

	minimumScheduleLead = 30 * time.Second
	maximumScheduleLead = 30 * 24 * time.Hour
	conflictWindow      = 60 * time.Second
	missedGrace         = 5 * time.Minute
	offlineAfter        = 2 * time.Minute
)

var (
	ErrInvalidConfiguration = errors.New("MVP store configuration is invalid")
	ErrHostAccessDenied     = errors.New("host access is denied")
	ErrScenarioInvalid      = errors.New("scenario is invalid")
	ErrScheduleInvalid      = errors.New("schedule is invalid")
	ErrScheduleMissed       = errors.New("schedule time has already passed")
	ErrScheduleConflict     = errors.New("schedule conflicts with an existing plan")
)

type Scenario struct {
	ID              string   `json:"id"`
	Version         string   `json:"version"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	RiskLevel       string   `json:"risk_level"`
	ExpectedEffects []string `json:"expected_effects"`
	CleanupRequired bool     `json:"cleanup_required"`
}

type Host struct {
	ID                 string     `json:"id"`
	EnvironmentID      string     `json:"environment_id"`
	RunnerID           string     `json:"runner_id,omitempty"`
	Name               string     `json:"name"`
	Status             string     `json:"status"`
	RunnerVersion      string     `json:"runner_version,omitempty"`
	LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
	AllowedScenarioIDs []string   `json:"allowed_scenario_ids"`
}

type ScheduleRequest struct {
	SchemaVersion     string `json:"schema_version"`
	HostID            string `json:"host_id"`
	ScenarioID        string `json:"scenario_id"`
	ScenarioVersion   string `json:"scenario_version"`
	ScheduledForLocal string `json:"scheduled_for_local"`
	TimeZone          string `json:"time_zone"`
}

type Schedule struct {
	SchemaVersion     string    `json:"schema_version"`
	ID                string    `json:"id"`
	HostID            string    `json:"host_id"`
	ScenarioID        string    `json:"scenario_id"`
	ScenarioVersion   string    `json:"scenario_version"`
	ScheduledForLocal string    `json:"scheduled_for_local"`
	ScheduledForUTC   time.Time `json:"scheduled_for_utc"`
	TimeZone          string    `json:"time_zone"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	JobID             string    `json:"job_id,omitempty"`
	CorrelationID     string    `json:"correlation_id,omitempty"`
}

type Config struct {
	WorkspaceID   string
	EnvironmentID string
	HostID        string
	RunnerID      string
	HostName      string
	RunnerVersion string
	Now           func() time.Time
}

type Service struct {
	mu            sync.Mutex
	workspaceID   string
	environmentID string
	hostID        string
	runnerID      string
	hostName      string
	runnerVersion string
	lastSeenAt    *time.Time
	now           func() time.Time
	scenarios     []Scenario
	schedules     map[string]Schedule
	scheduleOrder []string
}

func New(config Config) (*Service, error) {
	config.WorkspaceID = strings.ToLower(config.WorkspaceID)
	config.EnvironmentID = strings.ToLower(config.EnvironmentID)
	config.HostID = strings.ToLower(config.HostID)
	config.RunnerID = strings.ToLower(config.RunnerID)
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.HostName == "" || len(config.HostName) > 80 || config.WorkspaceID == "" ||
		config.EnvironmentID == "" || config.HostID == "" {
		return nil, ErrInvalidConfiguration
	}
	if config.RunnerVersion == "" {
		config.RunnerVersion = "0.1.0"
	}
	return &Service{
		workspaceID:   config.WorkspaceID,
		environmentID: config.EnvironmentID,
		hostID:        config.HostID,
		runnerID:      config.RunnerID,
		hostName:      config.HostName,
		runnerVersion: config.RunnerVersion,
		now:           config.Now,
		scenarios:     defaultScenarios(),
		schedules:     make(map[string]Schedule),
	}, nil
}

func (service *Service) Scenarios() []Scenario {
	service.mu.Lock()
	defer service.mu.Unlock()
	return cloneScenarios(service.scenarios)
}

func (service *Service) Hosts() []Host {
	service.mu.Lock()
	defer service.mu.Unlock()
	return []Host{service.hostLocked()}
}

func (service *Service) Authorize(hostID, scenarioID, scenarioVersion string) error {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.authorizeLocked(hostID, scenarioID, scenarioVersion)
}

func (service *Service) RecordRunnerSeen(runnerID, version string) error {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.runnerID == "" || strings.ToLower(runnerID) != service.runnerID {
		return ErrHostAccessDenied
	}
	if version != "" {
		if len(version) > 32 || strings.ContainsAny(version, "\r\n\t ") {
			return ErrInvalidConfiguration
		}
		service.runnerVersion = version
	}
	now := service.now().UTC()
	service.lastSeenAt = &now
	return nil
}

func (service *Service) CreateSchedule(request ScheduleRequest) (Schedule, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	request.HostID = strings.ToLower(request.HostID)
	request.ScenarioID = strings.ToLower(request.ScenarioID)
	if request.SchemaVersion != SchemaVersion ||
		service.authorizeLocked(request.HostID, request.ScenarioID, request.ScenarioVersion) != nil {
		return Schedule{}, ErrScheduleInvalid
	}
	location, err := approvedLocation(request.TimeZone)
	if err != nil {
		return Schedule{}, ErrScheduleInvalid
	}
	local, err := time.ParseInLocation("2006-01-02T15:04:05", request.ScheduledForLocal, location)
	if err != nil || local.Format("2006-01-02T15:04:05") != request.ScheduledForLocal {
		return Schedule{}, ErrScheduleInvalid
	}
	now := service.now().UTC()
	scheduledUTC := local.UTC()
	if scheduledUTC.Before(now.Add(minimumScheduleLead)) {
		return Schedule{}, ErrScheduleMissed
	}
	if scheduledUTC.After(now.Add(maximumScheduleLead)) {
		return Schedule{}, ErrScheduleInvalid
	}
	service.markMissedLocked(now)
	for _, existing := range service.schedules {
		if existing.Status != ScheduleStatusPlanned || existing.HostID != request.HostID {
			continue
		}
		difference := existing.ScheduledForUTC.Sub(scheduledUTC)
		if difference < 0 {
			difference = -difference
		}
		if difference < conflictWindow {
			return Schedule{}, ErrScheduleConflict
		}
	}
	id, err := randomID()
	if err != nil {
		return Schedule{}, err
	}
	schedule := Schedule{
		SchemaVersion:     SchemaVersion,
		ID:                id,
		HostID:            request.HostID,
		ScenarioID:        request.ScenarioID,
		ScenarioVersion:   request.ScenarioVersion,
		ScheduledForLocal: request.ScheduledForLocal,
		ScheduledForUTC:   scheduledUTC,
		TimeZone:          request.TimeZone,
		Status:            ScheduleStatusPlanned,
		CreatedAt:         now,
	}
	service.schedules[id] = schedule
	service.scheduleOrder = append(service.scheduleOrder, id)
	return schedule, nil
}

func (service *Service) Schedules() []Schedule {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.markMissedLocked(service.now().UTC())
	result := make([]Schedule, 0, len(service.scheduleOrder))
	for index := len(service.scheduleOrder) - 1; index >= 0; index-- {
		result = append(result, service.schedules[service.scheduleOrder[index]])
	}
	return result
}

func (service *Service) DispatchDue(queue *runqueue.Queue) error {
	service.mu.Lock()
	defer service.mu.Unlock()
	now := service.now().UTC()
	service.markMissedLocked(now)
	for _, id := range service.scheduleOrder {
		schedule := service.schedules[id]
		if schedule.Status != ScheduleStatusPlanned || now.Before(schedule.ScheduledForUTC) {
			continue
		}
		receipt, err := queue.Enqueue("schedule_"+strings.ReplaceAll(schedule.ID, "-", ""), runqueue.CreateRequest{
			SchemaVersion:   runqueue.SchemaVersion,
			EnvironmentID:   service.environmentID,
			HostID:          schedule.HostID,
			ScenarioID:      schedule.ScenarioID,
			ScenarioVersion: schedule.ScenarioVersion,
		})
		if err != nil {
			return err
		}
		schedule.Status = ScheduleStatusQueued
		schedule.JobID = receipt.JobID
		schedule.CorrelationID = receipt.CorrelationID
		service.schedules[id] = schedule
	}
	return nil
}

func (service *Service) hostLocked() Host {
	allowed := make([]string, 0, len(service.scenarios))
	for _, scenario := range service.scenarios {
		allowed = append(allowed, scenario.ID)
	}
	host := Host{
		ID:                 service.hostID,
		EnvironmentID:      service.environmentID,
		RunnerID:           service.runnerID,
		Name:               service.hostName,
		Status:             "offline",
		RunnerVersion:      service.runnerVersion,
		AllowedScenarioIDs: allowed,
	}
	if service.lastSeenAt != nil {
		lastSeen := *service.lastSeenAt
		host.LastSeenAt = &lastSeen
		if service.now().UTC().Sub(lastSeen) <= offlineAfter {
			host.Status = "online"
		}
	}
	return host
}

func (service *Service) authorizeLocked(hostID, scenarioID, scenarioVersion string) error {
	if strings.ToLower(hostID) != service.hostID {
		return ErrHostAccessDenied
	}
	for _, scenario := range service.scenarios {
		if scenario.ID == strings.ToLower(scenarioID) && scenario.Version == scenarioVersion {
			return nil
		}
	}
	return ErrScenarioInvalid
}

func (service *Service) markMissedLocked(now time.Time) {
	for id, schedule := range service.schedules {
		if schedule.Status == ScheduleStatusPlanned && now.After(schedule.ScheduledForUTC.Add(missedGrace)) {
			schedule.Status = ScheduleStatusMissed
			service.schedules[id] = schedule
		}
	}
}

func approvedLocation(name string) (*time.Location, error) {
	if name != "UTC" && name != "Europe/Istanbul" {
		return nil, ErrScheduleInvalid
	}
	return time.LoadLocation(name)
}

func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(buffer)
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

func cloneScenarios(source []Scenario) []Scenario {
	result := make([]Scenario, len(source))
	for index, scenario := range source {
		result[index] = scenario
		result[index].ExpectedEffects = append([]string(nil), scenario.ExpectedEffects...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func defaultScenarios() []Scenario {
	return []Scenario{
		{
			ID: "windows-process-marker", Version: "0.1.0", Name: "Windows Process Marker",
			Description: "Creates a harmless correlation marker in an approved child process.",
			RiskLevel:   "low", ExpectedEffects: []string{"One short-lived cmd.exe process", "One Sysmon process-create event"},
			CleanupRequired: true,
		},
		{
			ID: "windows-registry-run-key-canary", Version: "0.1.0", Name: "Registry Run Key Canary",
			Description: "Creates and removes one synthetic per-user Run key value.",
			RiskLevel:   "guarded", ExpectedEffects: []string{"One HKCU Run value", "Registry telemetry", "Mandatory value removal"},
			CleanupRequired: true,
		},
		{
			ID: "windows-scheduled-task-canary", Version: "0.1.0", Name: "Scheduled Task Canary",
			Description: "Registers and removes one non-executing synthetic scheduled task.",
			RiskLevel:   "guarded", ExpectedEffects: []string{"One disabled task definition", "Task Scheduler telemetry", "Mandatory task removal"},
			CleanupRequired: true,
		},
	}
}
