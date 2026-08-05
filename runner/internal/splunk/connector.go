package splunk

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ObserverUsername     = "prooflayer_observer"
	AllowedIndex         = "prooflayer_test"
	maximumResponseBytes = 64 * 1024
)

var (
	ErrInvalidConfig      = errors.New("invalid Splunk connector configuration")
	ErrUnauthorized       = errors.New("Splunk observer authentication failed")
	ErrForbidden          = errors.New("Splunk observer access was forbidden")
	ErrUnavailable        = errors.New("Splunk observer endpoint is unavailable")
	ErrUnexpectedResponse = errors.New("Splunk observer returned an unexpected response")
)

type Config struct {
	BaseURL  string
	Username string
	Password string
	TLS      *tls.Config
}

type Connector struct {
	baseURL  *url.URL
	username string
	password string
	client   *http.Client
}

func New(config Config, client *http.Client) (*Connector, error) {
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil || baseURL.Scheme != "https" || baseURL.Host == "" {
		return nil, ErrInvalidConfig
	}
	if baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, ErrInvalidConfig
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/")
	if config.Username != ObserverUsername || len(config.Password) < 16 {
		return nil, ErrInvalidConfig
	}

	if client == nil {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		if config.TLS != nil {
			tlsConfig = config.TLS.Clone()
			if tlsConfig.MinVersion == 0 {
				tlsConfig.MinVersion = tls.VersionTLS12
			}
		}
		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				Proxy:           http.ProxyFromEnvironment,
				TLSClientConfig: tlsConfig,
			},
		}
	}
	return &Connector{
		baseURL:  baseURL,
		username: config.Username,
		password: config.Password,
		client:   client,
	}, nil
}

func (connector *Connector) Check(ctx context.Context) error {
	deadlineContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	payload, err := connector.export(deadlineContext, "search index="+AllowedIndex+" | stats count")
	if err != nil || len(payload) == 0 || len(payload) > maximumResponseBytes {
		if err != nil {
			return err
		}
		return ErrUnexpectedResponse
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoded := 0
	for {
		var object map[string]any
		err := decoder.Decode(&object)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || len(object) == 0 {
			return ErrUnexpectedResponse
		}
		decoded++
	}
	if decoded == 0 {
		return ErrUnexpectedResponse
	}
	return nil
}

func (connector *Connector) export(ctx context.Context, search string) ([]byte, error) {
	endpoint := connector.baseURL.ResolveReference(&url.URL{Path: "/services/search/jobs/export"})
	form := url.Values{
		"search":      {search},
		"output_mode": {"json"},
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, ErrInvalidConfig
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(connector.username, connector.password)

	response, err := connector.client.Do(request)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusForbidden:
		return nil, ErrForbidden
	default:
		return nil, fmt.Errorf("%w: http_%d", ErrUnexpectedResponse, response.StatusCode)
	}

	reader := bufio.NewReader(io.LimitReader(response.Body, maximumResponseBytes+1))
	payload, err := io.ReadAll(reader)
	if err != nil || len(payload) > maximumResponseBytes {
		return nil, ErrUnexpectedResponse
	}
	return payload, nil
}
