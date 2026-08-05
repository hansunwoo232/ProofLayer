package splunk

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

const testPassword = "0123456789ABCDEF0123456789ABCDEF"

func TestCheckUsesFixedObserverIdentityAndIndex(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		username, password, ok := request.BasicAuth()
		if !ok || username != ObserverUsername || password != testPassword {
			return response(http.StatusUnauthorized, ""), nil
		}
		if request.Method != http.MethodPost || request.URL.Path != "/services/search/jobs/export" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("search") != "search index=prooflayer_test | stats count" {
			t.Errorf("search = %q", form.Get("search"))
		}
		if form.Get("output_mode") != "json" {
			t.Errorf("output mode = %q", form.Get("output_mode"))
		}
		return response(http.StatusOK, `{"preview":false,"result":{"count":"0"}}`+"\n"), nil
	})}

	connector, err := New(Config{BaseURL: "https://splunk.test:8089", Username: ObserverUsername, Password: testPassword}, client)
	if err != nil {
		t.Fatal(err)
	}
	if err := connector.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestCheckClassifiesInvalidAuthorization(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusUnauthorized, ""), nil
	})}
	connector, err := New(Config{BaseURL: "https://splunk.test:8089", Username: ObserverUsername, Password: testPassword}, client)
	if err != nil {
		t.Fatal(err)
	}
	if err := connector.Check(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckClassifiesUnreachableEndpointWithoutLeakingTransportError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed with secret-like internal detail")
	})}
	connector, err := New(Config{
		BaseURL:  "https://127.0.0.1:8089",
		Username: ObserverUsername,
		Password: testPassword,
	}, client)
	if err != nil {
		t.Fatal(err)
	}
	err = connector.Check(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "secret-like") {
		t.Fatal("transport detail leaked into connector error")
	}
}

func TestNewRejectsUnsafeConfiguration(t *testing.T) {
	tests := []Config{
		{BaseURL: "http://127.0.0.1:8089", Username: ObserverUsername, Password: testPassword},
		{BaseURL: "https://admin:secret@127.0.0.1:8089", Username: ObserverUsername, Password: testPassword},
		{BaseURL: "https://127.0.0.1:8089?token=secret", Username: ObserverUsername, Password: testPassword},
		{BaseURL: "https://127.0.0.1:8089", Username: "admin", Password: testPassword},
		{BaseURL: "https://127.0.0.1:8089", Username: ObserverUsername, Password: "short"},
	}
	for index, config := range tests {
		if _, err := New(config, nil); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func TestObserverPasswordFromEnvironment(t *testing.T) {
	oldValue, existed := os.LookupEnv(ObserverPasswordEnvironment)
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(ObserverPasswordEnvironment, oldValue)
		} else {
			_ = os.Unsetenv(ObserverPasswordEnvironment)
		}
	})
	_ = os.Unsetenv(ObserverPasswordEnvironment)
	if _, err := ObserverPasswordFromEnvironment(); !errors.Is(err, ErrCredentialUnavailable) {
		t.Fatalf("missing credential error = %v", err)
	}
	_ = os.Setenv(ObserverPasswordEnvironment, testPassword)
	password, err := ObserverPasswordFromEnvironment()
	if err != nil || password != testPassword {
		t.Fatal("environment credential was not loaded")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func response(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
