package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Request contains the confing options relative to a single request.
type Request struct {
	Method  string
	URL     *url.URL
	Timeout time.Duration
}

// RunnerOptions contains options relative to the runner.
type RunnerOptions struct {
	Requests      int
	Concurrency   int
	GlobalTimeout time.Duration
}

// Config represents the configuration of the runner.
// It must be validated using Config.Validate before usage.
type Config struct {
	Request       Request
	RunnerOptions RunnerOptions
}

// String returns an indented JSON representation of Config
// for debugging purposes.
func (cfg Config) String() string {
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return string(b)
}

// HTTPRequest returns a *http.Request created from Target. Returns any non-nil
// error that occurred.
func (cfg Config) HTTPRequest() (*http.Request, error) {
	if cfg.Request.URL == nil {
		return nil, errors.New("empty url")
	}
	rawURL := cfg.Request.URL.String()
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return nil, errors.New("bad url")
	}
	// TODO: handle body
	return http.NewRequest(cfg.Request.Method, rawURL, nil)
}

// Override returns a new Config based on cfg with overridden values from c.
// Only fields specified in options are replaced. Accepted options are limited
// to existing Fields, other values are silently ignored.
func (cfg Config) Override(c Config, fields ...string) Config {
	for _, field := range fields {
		switch field {
		case FieldMethod:
			cfg.Request.Method = c.Request.Method
		case FieldURL:
			cfg.Request.URL = c.Request.URL
		case FieldTimeout:
			cfg.Request.Timeout = c.Request.Timeout
		case FieldRequests:
			cfg.RunnerOptions.Requests = c.RunnerOptions.Requests
		case FieldConcurrency:
			cfg.RunnerOptions.Concurrency = c.RunnerOptions.Concurrency
		case FieldGlobalTimeout:
			cfg.RunnerOptions.GlobalTimeout = c.RunnerOptions.GlobalTimeout
		}
	}
	return cfg
}

// New returns a Config initialized with given parameters. The returned Config
// is not guaranteed to be safe: it must be validated using Config.Validate
// before usage.
func New(uri string, requests, concurrency int, requestTimeout, globalTimeout time.Duration) Config {
	// ignore err: a Config can be invalid at this point
	urlURL, _ := url.ParseRequestURI(uri)
	if urlURL == nil {
		urlURL = &url.URL{}
	}
	return Config{
		Request: Request{
			URL:     urlURL,
			Timeout: requestTimeout,
		},
		RunnerOptions: RunnerOptions{
			Requests:      requests,
			Concurrency:   concurrency,
			GlobalTimeout: globalTimeout,
		},
	}
}

// Validate returns the config and a not nil ErrInvalid if any of the fields provided by the user is not valid
func (cfg Config) Validate() error { //nolint:gocognit
	inputErrors := []error{}

	_, err := url.ParseRequestURI(cfg.Request.URL.String())
	if err != nil {
		inputErrors = append(inputErrors, fmt.Errorf("-url: %s is not a valid url", cfg.Request.URL.String()))
	}

	if cfg.RunnerOptions.Requests < 1 && cfg.RunnerOptions.Requests != -1 {
		inputErrors = append(inputErrors, fmt.Errorf("-requests: must be >= 0, we got %d", cfg.RunnerOptions.Requests))
	}

	if cfg.RunnerOptions.Concurrency < 1 && cfg.RunnerOptions.Concurrency != -1 {
		inputErrors = append(inputErrors, fmt.Errorf("-concurrency: must be > 0, we got %d", cfg.RunnerOptions.Concurrency))
	}

	if cfg.Request.Timeout < 0 {
		inputErrors = append(inputErrors, fmt.Errorf("-timeout: must be > 0, we got %d", cfg.Request.Timeout))
	}

	if cfg.RunnerOptions.GlobalTimeout < 0 {
		inputErrors = append(inputErrors, fmt.Errorf("-globalTimeout: must be > 0, we got %d", cfg.RunnerOptions.GlobalTimeout))
	}

	if len(inputErrors) > 0 {
		return &ErrInvalid{inputErrors}
	}
	return nil
}

// Default returns a default config that is safe to use.
func Default() Config {
	return defaultConfig
}
