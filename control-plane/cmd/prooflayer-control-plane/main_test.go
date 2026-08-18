package main

import (
	"crypto/ed25519"
	"encoding/base64"
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
