package correlation

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strings"
)

const Prefix = "PL-"

var (
	ErrGenerationFailed = errors.New("correlation ID generation failed")
	idPattern           = regexp.MustCompile(`^PL-[A-F0-9]{32}$`)
)

func Generate() (string, error) {
	return generate(rand.Reader)
}

func Valid(value string) bool {
	return idPattern.MatchString(value)
}

func generate(source io.Reader) (string, error) {
	random := make([]byte, 16)
	if _, err := io.ReadFull(source, random); err != nil {
		return "", ErrGenerationFailed
	}
	return Prefix + strings.ToUpper(hex.EncodeToString(random)), nil
}
