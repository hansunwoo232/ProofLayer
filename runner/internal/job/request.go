package job

import (
	"errors"
	"fmt"

	"github.com/hansunwoo232/ProofLayer/runner/internal/correlation"
	"github.com/hansunwoo232/ProofLayer/runner/internal/identity"
	"github.com/hansunwoo232/ProofLayer/runner/internal/scenario"
)

var ErrInvalidRequest = errors.New("invalid execution request")

type ExecutionRequest struct {
	CorrelationID   string         `json:"correlation_id"`
	EnvironmentID   string         `json:"environment_id"`
	HostID          string         `json:"host_id"`
	ScenarioID      string         `json:"scenario_id"`
	ScenarioVersion string         `json:"scenario_version"`
	Parameters      map[string]any `json:"parameters"`
}

func (request ExecutionRequest) Validate(runnerIdentity identity.RunnerIdentity, catalog scenario.Catalog) (scenario.Definition, error) {
	if !correlation.Valid(request.CorrelationID) {
		return scenario.Definition{}, invalid("correlation_id")
	}
	if !runnerIdentity.Authorizes(request.EnvironmentID, request.HostID) {
		return scenario.Definition{}, invalid("identity_binding")
	}
	definition, err := catalog.Resolve(request.ScenarioID, request.ScenarioVersion)
	if err != nil {
		return scenario.Definition{}, fmt.Errorf("%w: scenario_allowlist", ErrInvalidRequest)
	}
	if len(request.Parameters) != 0 {
		return scenario.Definition{}, invalid("parameters")
	}
	return definition, nil
}

func invalid(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidRequest, field)
}
