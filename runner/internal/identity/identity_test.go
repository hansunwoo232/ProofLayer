package identity

import (
	"errors"
	"testing"
	"time"
)

func validIdentity(now time.Time) RunnerIdentity {
	return RunnerIdentity{
		SchemaVersion: SchemaVersion,
		RunnerID:      "550e8400-e29b-41d4-a716-446655440000",
		EnvironmentID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
		HostID:        "6ba7b811-9dad-41d1-80b4-00c04fd430c8",
		IdentityKeyID: "runner-key-01",
		RegisteredAt:  now.Add(-time.Minute),
		State:         StateActive,
	}
}

func TestRunnerIdentityValidate(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	if err := validIdentity(now).Validate(now); err != nil {
		t.Fatalf("valid identity rejected: %v", err)
	}
}

func TestRunnerIdentityRejectsInvalidBindings(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	tests := []RunnerIdentity{
		func() RunnerIdentity { value := validIdentity(now); value.SchemaVersion = "2.0"; return value }(),
		func() RunnerIdentity { value := validIdentity(now); value.RunnerID = "not-a-uuid"; return value }(),
		func() RunnerIdentity { value := validIdentity(now); value.IdentityKeyID = "bad key"; return value }(),
		func() RunnerIdentity {
			value := validIdentity(now)
			value.RegisteredAt = now.Add(2 * time.Minute)
			return value
		}(),
		func() RunnerIdentity { value := validIdentity(now); value.State = "unknown"; return value }(),
	}

	for index, identity := range tests {
		if err := identity.Validate(now); !errors.Is(err, ErrInvalidIdentity) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func TestRunnerIdentityAuthorizationIsHostBoundAndFailClosed(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	identity := validIdentity(now)
	if !identity.Authorizes(identity.EnvironmentID, identity.HostID) {
		t.Fatal("active exact binding was not authorized")
	}
	if identity.Authorizes(identity.EnvironmentID, "550e8400-e29b-41d4-a716-446655440000") {
		t.Fatal("different host was authorized")
	}
	identity.State = StateRevoked
	if identity.Authorizes(identity.EnvironmentID, identity.HostID) {
		t.Fatal("revoked identity was authorized")
	}
}
