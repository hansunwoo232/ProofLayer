package localauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	minimumPasswordBytes = 12
	maximumPasswordBytes = 128
)

var (
	ErrPasswordPolicy = errors.New("password does not satisfy local policy")
	ErrPasswordHash   = errors.New("password hash is invalid")
)

type PasswordParameters struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltBytes   uint32
	KeyBytes    uint32
}

func DefaultPasswordParameters() PasswordParameters {
	return PasswordParameters{
		MemoryKiB:   64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltBytes:   16,
		KeyBytes:    32,
	}
}

func HashPassword(password string, parameters PasswordParameters) (string, error) {
	if len(password) < minimumPasswordBytes || len(password) > maximumPasswordBytes {
		return "", ErrPasswordPolicy
	}
	if err := parameters.validate(); err != nil {
		return "", err
	}
	salt := make([]byte, parameters.SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey(
		[]byte(password),
		salt,
		parameters.Iterations,
		parameters.MemoryKiB,
		parameters.Parallelism,
		parameters.KeyBytes,
	)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		parameters.MemoryKiB,
		parameters.Iterations,
		parameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parameters, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		parameters.Iterations,
		parameters.MemoryKiB,
		parameters.Parallelism,
		uint32(len(expected)),
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parsePasswordHash(encoded string) (PasswordParameters, []byte, []byte, error) {
	var parameters PasswordParameters
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return parameters, nil, nil, ErrPasswordHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return parameters, nil, nil, ErrPasswordHash
	}
	var parallelism uint32
	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&parameters.MemoryKiB,
		&parameters.Iterations,
		&parallelism,
	); err != nil || parallelism > 255 {
		return parameters, nil, nil, ErrPasswordHash
	}
	if parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", parameters.MemoryKiB, parameters.Iterations, parallelism) {
		return parameters, nil, nil, ErrPasswordHash
	}
	parameters.Parallelism = uint8(parallelism)
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return parameters, nil, nil, ErrPasswordHash
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return parameters, nil, nil, ErrPasswordHash
	}
	parameters.SaltBytes = uint32(len(salt))
	parameters.KeyBytes = uint32(len(key))
	if err := parameters.validate(); err != nil {
		return parameters, nil, nil, ErrPasswordHash
	}
	return parameters, salt, key, nil
}

func (parameters PasswordParameters) validate() error {
	if parameters.MemoryKiB < 8*1024 || parameters.MemoryKiB > 256*1024 ||
		parameters.Iterations < 1 || parameters.Iterations > 10 ||
		parameters.Parallelism < 1 || parameters.Parallelism > 8 ||
		parameters.SaltBytes < 16 || parameters.SaltBytes > 64 ||
		parameters.KeyBytes < 16 || parameters.KeyBytes > 64 {
		return ErrPasswordHash
	}
	return nil
}
