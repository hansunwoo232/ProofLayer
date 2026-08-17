package executor

import (
	"reflect"
	"testing"
)

func TestScheduledTaskCanaryNameIsFixedAndCorrelationBound(t *testing.T) {
	taskName, err := scheduledTaskCanaryName("PL-0123456789ABCDEF0123456789ABCDEF")
	if err != nil {
		t.Fatal(err)
	}
	if taskName != "ProofLayer_0123456789ABCDEF0123456789ABCDEF" {
		t.Fatalf("task name = %q", taskName)
	}
}

func TestScheduledTaskCanaryRejectsInvalidCorrelationID(t *testing.T) {
	if _, err := scheduledTaskCanaryName(`..\user-controlled`); err == nil {
		t.Fatal("invalid correlation ID accepted")
	}
}

func TestScheduledTaskCanaryUsesOnlyFixedArguments(t *testing.T) {
	taskName := "ProofLayer_0123456789ABCDEF0123456789ABCDEF"
	wantCreate := []string{
		"/Create",
		"/TN", taskName,
		"/TR", `C:\Windows\System32\cmd.exe /d /c exit 0`,
		"/SC", "ONLOGON",
		"/RL", "LIMITED",
		"/F",
	}
	if got := scheduledTaskCreateArguments(taskName); !reflect.DeepEqual(got, wantCreate) {
		t.Fatalf("create arguments = %#v", got)
	}
	wantDelete := []string{"/Delete", "/TN", taskName, "/F"}
	if got := scheduledTaskDeleteArguments(taskName); !reflect.DeepEqual(got, wantDelete) {
		t.Fatalf("delete arguments = %#v", got)
	}
}
