package pipeline

import "testing"

func TestDisplayLanguageIsExplicitAndNotColorOnly(t *testing.T) {
	tests := []struct {
		status StageStatus
		label  string
		tone   string
		symbol string
	}{
		{StatusPassed, "PASS", "positive", "check"},
		{StatusFailed, "FAIL", "critical", "x"},
		{StatusNotTested, "NOT TESTED", "neutral", "minus"},
		{StatusDegraded, "DEGRADED", "warning", "warning"},
		{StatusRunning, "RUNNING", "informative", "progress"},
		{StatusPending, "PENDING", "neutral", "clock"},
	}
	for _, test := range tests {
		display, err := Display(test.status)
		if err != nil {
			t.Fatal(err)
		}
		if display.Label != test.label || display.Tone != test.tone || display.Symbol != test.symbol {
			t.Fatalf("Display(%s) = %+v", test.status, display)
		}
	}
}

func TestDisplayRejectsUnknownStatus(t *testing.T) {
	if _, err := Display("unknown"); err == nil {
		t.Fatal("unknown status was accepted")
	}
}
