package correlation

import (
	"errors"
	"testing"
)

func TestGenerateProducesCanonicalUniqueIDs(t *testing.T) {
	seen := make(map[string]struct{}, 128)
	for range 128 {
		value, err := Generate()
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if !Valid(value) {
			t.Fatalf("invalid generated value %q", value)
		}
		if _, exists := seen[value]; exists {
			t.Fatalf("duplicate generated value %q", value)
		}
		seen[value] = struct{}{}
	}
}

func TestGenerateFailsClosedWhenRandomSourceFails(t *testing.T) {
	_, err := generate(errorReader{})
	if !errors.Is(err, ErrGenerationFailed) {
		t.Fatalf("error = %v", err)
	}
}

func TestValidRejectsNonCanonicalIDs(t *testing.T) {
	invalid := []string{
		"",
		"PL-0123",
		"pl-0123456789abcdef0123456789abcdef",
		"PL-0123456789abcdef0123456789abcdef",
		"PL-0123456789ABCDEF0123456789ABCDEG",
		"PL-0123456789ABCDEF0123456789ABCDEF-extra",
	}
	for _, value := range invalid {
		if Valid(value) {
			t.Fatalf("Valid(%q) = true", value)
		}
	}
}

type errorReader struct{}

func (errorReader) Read(buffer []byte) (int, error) {
	return 0, errors.New("random source unavailable")
}
