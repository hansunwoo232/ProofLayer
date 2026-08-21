package localauth

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"
)

const testPassword = "correct horse battery staple"

func testService(t *testing.T, now *time.Time) *Service {
	t.Helper()
	random := bytes.NewReader(bytes.Repeat([]byte{0x5a}, 512))
	service, err := New(Config{
		Workspace: Workspace{
			ID:   "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
			Slug: "prooflayer-lab",
			Name: "ProofLayer Lab",
		},
		User: User{
			ID:          "7ba7b811-9dad-41d1-80b4-00c04fd430c8",
			WorkspaceID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
			Email:       "Admin@ProofLayer.Local ",
			DisplayName: "Local Administrator",
			Role:        RoleAdmin,
			Status:      StatusActive,
		},
		Password:           testPassword,
		PasswordParameters: testPasswordParameters(),
		IdleTimeout:        30 * time.Minute,
		AbsoluteTimeout:    time.Hour,
		Now:                func() time.Time { return *now },
		Random:             random,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestAuthenticateCreatesWorkspaceBoundPrincipal(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	service := testService(t, &now)
	principal, err := service.Authenticate(" ADMIN@prooflayer.local", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if principal.UserID != "7ba7b811-9dad-41d1-80b4-00c04fd430c8" ||
		principal.WorkspaceID != service.Workspace().ID || principal.Role != RoleAdmin {
		t.Fatalf("principal = %+v", principal)
	}
	for _, test := range []struct{ email, password string }{
		{email: "unknown@prooflayer.local", password: testPassword},
		{email: "admin@prooflayer.local", password: "wrong password value"},
	} {
		if _, err := service.Authenticate(test.email, test.password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("credentials error = %v", err)
		}
	}
}

func TestSessionStoresOnlyDigestAndHonorsIdleExpiry(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	service := testService(t, &now)
	principal, _ := service.Authenticate("admin@prooflayer.local", testPassword)
	token, err := service.CreateSession(principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) < 40 {
		t.Fatalf("token length = %d", len(token))
	}
	digest := sha256.Sum256([]byte(token))
	stored, ok := service.sessions[digest]
	if !ok || stored.TokenDigest != digest || strings.Contains(stored.ID, token) {
		t.Fatalf("stored session = %+v", stored)
	}
	if _, err := service.VerifySession(token); err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * time.Minute)
	if _, err := service.VerifySession(token); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expired session error = %v", err)
	}
}

func TestSessionAbsoluteExpiryAndLogout(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	service := testService(t, &now)
	principal, _ := service.Authenticate("admin@prooflayer.local", testPassword)
	token, _ := service.CreateSession(principal)
	for _, advance := range []time.Duration{25 * time.Minute, 25 * time.Minute} {
		now = now.Add(advance)
		if _, err := service.VerifySession(token); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(11 * time.Minute)
	if _, err := service.VerifySession(token); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("absolute expiry error = %v", err)
	}

	now = time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	service = testService(t, &now)
	principal, _ = service.Authenticate("admin@prooflayer.local", testPassword)
	token, _ = service.CreateSession(principal)
	service.RevokeSession(token)
	if _, err := service.VerifySession(token); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("revoked session error = %v", err)
	}
}
