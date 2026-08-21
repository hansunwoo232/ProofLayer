package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"testing"
)

func TestLoadSigningPrivateKeyIsStableForConfiguredSeed(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	encoded := base64.RawURLEncoding.EncodeToString(seed)
	first, err := loadSigningPrivateKey(encoded)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadSigningPrivateKey(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Equal(second) {
		t.Fatal("configured signing seed did not produce a stable key")
	}
}

func TestLoadSigningPrivateKeyRejectsInvalidSeed(t *testing.T) {
	if _, err := loadSigningPrivateKey("too-short"); err == nil {
		t.Fatal("invalid seed was accepted")
	}
}

func TestLocalAuthenticationIsOptInAndClearsBootstrapPassword(t *testing.T) {
	t.Setenv("PROOFLAYER_LOCAL_ADMIN_PASSWORD", "")
	service, err := localAuthenticationFromEnvironment()
	if err != nil || service != nil {
		t.Fatalf("disabled local authentication = %v, error = %v", service, err)
	}

	t.Setenv("PROOFLAYER_LOCAL_ADMIN_PASSWORD", "correct horse battery staple")
	t.Setenv("PROOFLAYER_LOCAL_ADMIN_EMAIL", "Operator@ProofLayer.Local")
	service, err = localAuthenticationFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if service == nil || service.Workspace().ID != localWorkspaceID {
		t.Fatalf("local authentication = %+v", service)
	}
	if _, present := os.LookupEnv("PROOFLAYER_LOCAL_ADMIN_PASSWORD"); present {
		t.Fatal("bootstrap password remained in the process environment")
	}
	principal, err := service.Authenticate("operator@prooflayer.local", "correct horse battery staple")
	if err != nil || principal.WorkspaceID != localWorkspaceID {
		t.Fatalf("principal = %+v, error = %v", principal, err)
	}
}

func TestLocalAuthenticationRejectsWeakBootstrapPassword(t *testing.T) {
	t.Setenv("PROOFLAYER_LOCAL_ADMIN_PASSWORD", "too-short")
	if _, err := localAuthenticationFromEnvironment(); err == nil {
		t.Fatal("weak bootstrap password was accepted")
	}
}
