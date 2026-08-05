package identity

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

const SchemaVersion = "1.0"

var (
	ErrInvalidIdentity = errors.New("invalid runner identity")
	uuidPattern        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	keyIDPattern       = regexp.MustCompile(`^[A-Za-z0-9._-]{1,80}$`)
)

type State string

const (
	StateActive  State = "active"
	StateRevoked State = "revoked"
)

type RunnerIdentity struct {
	SchemaVersion string    `json:"schema_version"`
	RunnerID      string    `json:"runner_id"`
	EnvironmentID string    `json:"environment_id"`
	HostID        string    `json:"host_id"`
	IdentityKeyID string    `json:"identity_key_id"`
	RegisteredAt  time.Time `json:"registered_at"`
	State         State     `json:"state"`
}

func (identity RunnerIdentity) Validate(now time.Time) error {
	if identity.SchemaVersion != SchemaVersion {
		return invalid("schema_version")
	}
	for name, value := range map[string]string{
		"runner_id":      identity.RunnerID,
		"environment_id": identity.EnvironmentID,
		"host_id":        identity.HostID,
	} {
		if !uuidPattern.MatchString(value) {
			return invalid(name)
		}
	}
	if !keyIDPattern.MatchString(identity.IdentityKeyID) {
		return invalid("identity_key_id")
	}
	if identity.RegisteredAt.IsZero() || identity.RegisteredAt.After(now.Add(time.Minute)) {
		return invalid("registered_at")
	}
	if identity.State != StateActive && identity.State != StateRevoked {
		return invalid("state")
	}
	return nil
}

func (identity RunnerIdentity) Authorizes(environmentID, hostID string) bool {
	return identity.State == StateActive &&
		identity.EnvironmentID == environmentID &&
		identity.HostID == hostID
}

func invalid(field string) error {
	return fmt.Errorf("%w: %s", ErrInvalidIdentity, field)
}
