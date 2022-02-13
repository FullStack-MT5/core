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
	Method string
	URL    *url.URL
	Header http.Header
}

// HTTP generates a *http.Request based on Request and returns it
// or any non-nil error that occurred.
func (r Request) HTTP() (*http.Request, error) {
	if r.URL == nil {
		return nil, errors.New("empty url")
	}
	rawURL := r.URL.String()
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return nil, errors.New("bad url")
	}
	// TODO: handle body
	req, err := http.NewRequest(r.Method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header = r.Header
	return req, nil
}

// Runner contains options relative to the runner.
type Runner struct {
	Requests       int
	Concurrency    int
	Interval       time.Duration
	RequestTimeout time.Duration
	GlobalTimeout  time.Duration
}

// Global represents the global configuration of the runner.
// It must be validated using Global.Validate before usage.
type Global struct {
	Request Request
	Runner  Runner
}

// String returns an indented JSON representation of Config
// for debugging purposes.
func (cfg Global) String() string {
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return string(b)
}

// Override returns a new Config based on cfg with overridden values from c.
// Only fields specified in options are replaced. Accepted options are limited
// to existing Fields, other values are silently ignored.
func (cfg Global) Override(c Global, fields ...string) Global {
	for _, field := range fields {
		switch field {
		case FieldMethod:
			cfg.Request.Method = c.Request.Method
		case FieldURL:
			cfg.Request.URL = c.Request.URL
		case FieldHeader:
			cfg.overrideHeader(c.Request.Header)
		case FieldRequests:
			cfg.Runner.Requests = c.Runner.Requests
		case FieldConcurrency:
			cfg.Runner.Concurrency = c.Runner.Concurrency
		case FieldInterval:
			cfg.Runner.Interval = c.Runner.Interval
		case FieldRequestTimeout:
			cfg.Runner.RequestTimeout = c.Runner.RequestTimeout
		case FieldGlobalTimeout:
			cfg.Runner.GlobalTimeout = c.Runner.GlobalTimeout
		}
	}
	return cfg
}

func (cfg *Global) overrideHeader(newHeader http.Header) {
	if newHeader == nil {
		return
	}
	if cfg.Request.Header == nil {
		cfg.Request.Header = http.Header{}
	}
	for k, v := range newHeader {
		cfg.Request.Header[k] = v
	}
}

// WithURL sets the current Config to the parsed *url.URL from rawURL
// and returns it. Any errors is discarded as a Config can be invalid
// until Config.Validate is called. The url is guaranteed not to be nil.
func (cfg Global) WithURL(rawURL string) Global {
	// ignore err: a Config can be invalid at this point
	urlURL, _ := url.ParseRequestURI(rawURL)
	if urlURL == nil {
		urlURL = &url.URL{}
	}
	cfg.Request.URL = urlURL
	return cfg
}

// Validate returns the config and a not nil ErrInvalid if any of the fields provided by the user is not valid
func (cfg Global) Validate() error { //nolint:gocognit
	inputErrors := []error{}

	if cfg.Request.URL == nil {
		inputErrors = append(inputErrors, errors.New("-url: missing url"))
	} else if _, err := url.ParseRequestURI(cfg.Request.URL.String()); err != nil {
		inputErrors = append(inputErrors, fmt.Errorf("-url: %s is not a valid url", cfg.Request.URL.String()))
	}

	if cfg.Runner.Requests < 1 && cfg.Runner.Requests != -1 {
		inputErrors = append(inputErrors, fmt.Errorf("-requests: must be >= 0, we got %d", cfg.Runner.Requests))
	}

	if cfg.Runner.Concurrency < 1 && cfg.Runner.Concurrency != -1 {
		inputErrors = append(inputErrors, fmt.Errorf("-concurrency: must be > 0, we got %d", cfg.Runner.Concurrency))
	}

	if cfg.Runner.Interval < 0 {
		inputErrors = append(inputErrors, fmt.Errorf("-interval: must be > 0, we got %d", cfg.Runner.Interval))
	}

	if cfg.Runner.RequestTimeout < 0 {
		inputErrors = append(inputErrors, fmt.Errorf("-timeout: must be > 0, we got %d", cfg.Runner.RequestTimeout))
	}

	if cfg.Runner.GlobalTimeout < 0 {
		inputErrors = append(inputErrors, fmt.Errorf("-globalTimeout: must be > 0, we got %d", cfg.Runner.GlobalTimeout))
	}

	if len(inputErrors) > 0 {
		return &ErrInvalid{inputErrors}
	}
	return nil
}

// Default returns a default config that is safe to use.
func Default() Global {
	return defaultConfig
}
