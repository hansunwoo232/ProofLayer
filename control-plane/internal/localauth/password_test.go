package localauth

import (
	"errors"
	"strings"
	"testing"
)

func testPasswordParameters() PasswordParameters {
	return PasswordParameters{
		MemoryKiB:   8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltBytes:   16,
		KeyBytes:    32,
	}
}

func TestPasswordHashRoundTrip(t *testing.T) {
	encoded, err := HashPassword("correct horse battery staple", testPasswordParameters())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$") || strings.Contains(encoded, "correct horse") {
		t.Fatalf("unsafe or unexpected hash = %q", encoded)
	}
	matched, err := VerifyPassword("correct horse battery staple", encoded)
	if err != nil || !matched {
		t.Fatalf("matched = %v, error = %v", matched, err)
	}
	matched, err = VerifyPassword("incorrect password", encoded)
	if err != nil || matched {
		t.Fatalf("wrong password matched = %v, error = %v", matched, err)
	}
}

func TestPasswordPolicyAndHashParsingFailClosed(t *testing.T) {
	if _, err := HashPassword("too-short", testPasswordParameters()); !errors.Is(err, ErrPasswordPolicy) {
		t.Fatalf("short password error = %v", err)
	}
	for _, encoded := range []string{
		"",
		"$argon2i$v=19$m=8192,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2U",
		"$argon2id$v=19$m=999999,t=1,p=1$c2FsdHNhbHRzYWx0c2FsdA$a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2U",
		"$argon2id$v=19$m=8192,t=1,p=1junk$c2FsdHNhbHRzYWx0c2FsdA$a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2U",
	} {
		if _, err := VerifyPassword("irrelevant password", encoded); !errors.Is(err, ErrPasswordHash) {
			t.Fatalf("hash %q error = %v", encoded, err)
		}
	}
}
