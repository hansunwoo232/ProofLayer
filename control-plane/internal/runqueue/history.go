package runqueue

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalidHistoryQuery = errors.New("history query is invalid")

type HistoryQuery struct {
	HostID     string
	ScenarioID string
	From       *time.Time
	To         *time.Time
	Page       int
	PageSize   int
}

type HistoryItem struct {
	JobID           string    `json:"job_id"`
	CorrelationID   string    `json:"correlation_id"`
	HostID          string    `json:"host_id"`
	ScenarioID      string    `json:"scenario_id"`
	ScenarioVersion string    `json:"scenario_version"`
	Status          JobStatus `json:"status"`
	Outcome         string    `json:"outcome"`
	RequestedAt     time.Time `json:"requested_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Terminal        bool      `json:"terminal"`
}

type HistoryPage struct {
	SchemaVersion string        `json:"schema_version"`
	Items         []HistoryItem `json:"items"`
	Page          int           `json:"page"`
	PageSize      int           `json:"page_size"`
	TotalItems    int           `json:"total_items"`
	TotalPages    int           `json:"total_pages"`
}

func (queue *Queue) History(query HistoryQuery) (HistoryPage, error) {
	query.HostID = strings.ToLower(query.HostID)
	query.ScenarioID = strings.ToLower(query.ScenarioID)
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}
	if query.Page < 1 || query.PageSize < 1 || query.PageSize > 50 ||
		(query.From != nil && query.To != nil && query.From.After(*query.To)) {
		return HistoryPage{}, ErrInvalidHistoryQuery
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	queue.pruneExpiredLocked(queue.now().UTC())

	candidates := queue.historyOrder
	if query.HostID != "" {
		candidates = queue.historyByHost[query.HostID]
	} else if query.ScenarioID != "" {
		candidates = queue.historyByScenario[query.ScenarioID]
	}
	items := make([]HistoryItem, 0, len(candidates))
	for index := len(candidates) - 1; index >= 0; index-- {
		record, ok := queue.lifecycle[candidates[index]]
		if !ok || (query.HostID != "" && record.Job.HostID != query.HostID) ||
			(query.ScenarioID != "" && record.Job.ScenarioID != query.ScenarioID) ||
			(query.From != nil && record.Job.RequestedAt.Before(query.From.UTC())) ||
			(query.To != nil && record.Job.RequestedAt.After(query.To.UTC())) {
			continue
		}
		items = append(items, historyItem(record))
	}
	total := len(items)
	start := (query.Page - 1) * query.PageSize
	if start > total {
		start = total
	}
	end := start + query.PageSize
	if end > total {
		end = total
	}
	pageItems := append([]HistoryItem(nil), items[start:end]...)
	totalPages := 0
	if total > 0 {
		totalPages = (total + query.PageSize - 1) / query.PageSize
	}
	return HistoryPage{
		SchemaVersion: SchemaVersion, Items: pageItems, Page: query.Page,
		PageSize: query.PageSize, TotalItems: total, TotalPages: totalPages,
	}, nil
}

func historyItem(record *lifecycleRecord) HistoryItem {
	outcome := "in_progress"
	if record.Terminal {
		if record.Status == JobStatusCompleted {
			outcome = "passed"
		} else {
			outcome = "failed"
		}
	}
	return HistoryItem{
		JobID: record.Job.JobID, CorrelationID: record.Job.CorrelationID,
		HostID: record.Job.HostID, ScenarioID: record.Job.ScenarioID,
		ScenarioVersion: record.Job.ScenarioVersion, Status: record.Status,
		Outcome: outcome, RequestedAt: record.Job.RequestedAt,
		UpdatedAt: record.UpdatedAt, Terminal: record.Terminal,
	}
}
