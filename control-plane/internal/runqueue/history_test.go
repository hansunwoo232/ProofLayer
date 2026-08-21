package runqueue

import (
	"errors"
	"testing"
	"time"
)

func TestHistoryFiltersAndPaginatesNewestFirst(t *testing.T) {
	queue, _ := newTestQueue(t, 8)
	scenarios := []string{
		"windows-process-marker",
		"windows-registry-run-key-canary",
		"windows-process-marker",
	}
	for index, scenario := range scenarios {
		request := validRequest()
		request.ScenarioID = scenario
		if _, err := queue.Enqueue([]string{
			"history_test_0123456789ABCDEF0001",
			"history_test_0123456789ABCDEF0002",
			"history_test_0123456789ABCDEF0003",
		}[index], request); err != nil {
			t.Fatal(err)
		}
	}

	page, err := queue.History(HistoryQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalItems != 3 || page.TotalPages != 2 || len(page.Items) != 2 ||
		page.Items[0].ScenarioID != "windows-process-marker" || page.Items[1].ScenarioID != "windows-registry-run-key-canary" {
		t.Fatalf("history page = %+v", page)
	}
	filtered, err := queue.History(HistoryQuery{ScenarioID: "windows-registry-run-key-canary", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.TotalItems != 1 || filtered.Items[0].ScenarioID != "windows-registry-run-key-canary" {
		t.Fatalf("filtered history = %+v", filtered)
	}
	empty, err := queue.History(HistoryQuery{HostID: "00000000-0000-4000-8000-000000000000", Page: 1, PageSize: 20})
	if err != nil || empty.TotalItems != 0 || len(empty.Items) != 0 {
		t.Fatalf("empty history = %+v, error = %v", empty, err)
	}
}

func TestHistoryRejectsInvalidDateAndPagination(t *testing.T) {
	queue, _ := newTestQueue(t, 4)
	from := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	to := from.Add(-time.Hour)
	for _, query := range []HistoryQuery{
		{Page: -1, PageSize: 20},
		{Page: 1, PageSize: 51},
		{Page: 1, PageSize: 20, From: &from, To: &to},
	} {
		if _, err := queue.History(query); !errors.Is(err, ErrInvalidHistoryQuery) {
			t.Fatalf("query %+v error = %v", query, err)
		}
	}
}
