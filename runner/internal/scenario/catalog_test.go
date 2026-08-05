package scenario

import (
	"errors"
	"testing"
)

func TestBuiltInCatalogContainsOnlyApprovedScenario(t *testing.T) {
	catalog := BuiltInCatalog()
	if catalog.Len() != 1 {
		t.Fatalf("allowlist size = %d, want 1", catalog.Len())
	}

	definition, err := catalog.Resolve("windows-process-marker", "0.1.0")
	if err != nil {
		t.Fatalf("resolve approved scenario: %v", err)
	}
	if definition.Handler != "builtin.emit_process_marker" {
		t.Fatalf("handler = %q", definition.Handler)
	}
	if definition.CleanupHandler != "builtin.verify_no_artifacts" {
		t.Fatalf("cleanup handler = %q", definition.CleanupHandler)
	}
}

func TestCatalogRejectsUnknownScenarioAndVersion(t *testing.T) {
	catalog := BuiltInCatalog()
	tests := []struct {
		id      string
		version string
	}{
		{id: "arbitrary-command", version: "0.1.0"},
		{id: "windows-process-marker", version: "0.1.1"},
		{id: "", version: ""},
	}

	for _, test := range tests {
		_, err := catalog.Resolve(test.id, test.version)
		if !errors.Is(err, ErrNotAllowlisted) {
			t.Fatalf("Resolve(%q, %q) error = %v", test.id, test.version, err)
		}
	}
}

func TestCatalogReturnsDefensiveCopies(t *testing.T) {
	catalog := BuiltInCatalog()
	first, err := catalog.Resolve("windows-process-marker", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	first.RequiredFields[0] = "modified"

	second, err := catalog.Resolve("windows-process-marker", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if second.RequiredFields[0] == "modified" {
		t.Fatal("catalog definition was mutated through returned slice")
	}
}
