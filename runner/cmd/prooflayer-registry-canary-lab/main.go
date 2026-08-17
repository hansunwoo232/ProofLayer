// prooflayer-registry-canary-lab is an isolated-lab harness for one fixed
// built-in Registry Run Key canary. It accepts no command, path, value, payload,
// argument, or scenario input.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hansunwoo232/ProofLayer/runner/internal/audit"
	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
	"github.com/hansunwoo232/ProofLayer/runner/internal/executor"
	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	"github.com/hansunwoo232/ProofLayer/runner/internal/job"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

const (
	labRunnerID      = "550e8400-e29b-41d4-a716-446655440000"
	labEnvironmentID = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	labHostID        = "6ba7b811-9dad-41d1-80b4-00c04fd430c8"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "this fixed registry canary harness accepts no arguments")
		os.Exit(2)
	}
	correlationID, err := correlation.Generate()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dataRoot := os.Getenv("ProgramData")
	if dataRoot == "" {
		dataRoot = os.TempDir()
	}
	auditDirectory := filepath.Join(dataRoot, "ProofLayer")
	if err := os.MkdirAll(auditDirectory, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "create audit directory:", err)
		os.Exit(1)
	}
	recorder, err := audit.NewFileRecorder(filepath.Join(auditDirectory, "runner-audit.jsonl"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	runnerIdentity := identity.RunnerIdentity{
		SchemaVersion: identity.SchemaVersion,
		RunnerID:      labRunnerID,
		EnvironmentID: labEnvironmentID,
		HostID:        labHostID,
		IdentityKeyID: "isolated-lab-key",
		RegisteredAt:  now.Add(-time.Minute),
		State:         identity.StateActive,
	}
	runner := executor.New(runnerIdentity, scenario.BuiltInCatalog(), recorder)
	result := runner.Execute(context.Background(), job.ExecutionRequest{
		CorrelationID:   correlationID,
		EnvironmentID:   labEnvironmentID,
		HostID:          labHostID,
		ScenarioID:      "windows-registry-run-key-canary",
		ScenarioVersion: "0.1.0",
		Parameters:      map[string]any{},
	})

	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if result.Status != executor.StatusPassed || result.CleanupStatus != executor.StatusPassed {
		os.Exit(1)
	}
}
