package file

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/benchttp/runner/config"
)

// Parse parses a benchttp runner config file into a config.Config
// and returns it or the first non-nil error occurring in the process.
func Parse(cfgpath string) (cfg config.Config, err error) {
	b, err := os.ReadFile(cfgpath)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return cfg, errWithDetails(ErrFileNotFound, cfgpath)
	default:
		return cfg, errWithDetails(ErrFileRead, cfgpath, err)
	}

	ext := extension(path.Ext(cfgpath))
	parser, err := newParser(ext)
	if err != nil {
		return cfg, errWithDetails(ErrFileExt, ext, err)
	}

	var rawCfg unmarshaledConfig
	if err = parser.parse(b, &rawCfg); err != nil {
		return cfg, errWithDetails(ErrParse, cfgpath, err)
	}

	cfg, err = parseRawConfig(rawCfg)
	if err != nil {
		return cfg, errWithDetails(ErrParse, cfgpath, err)
	}

	return
}

// parseRawConfig parses an input raw config as a config.Config and returns it
// or the first non-nil error occurring in the process.
func parseRawConfig(raw unmarshaledConfig) (config.Config, error) { //nolint:gocognit // acceptable complexity for a parsing func
	cfg := config.Config{}
	fields := make([]string, 0, 9)

	if method := raw.Request.Method; method != nil {
		cfg.Request.Method = *method
		fields = append(fields, config.FieldMethod)
	}

	if rawURL := raw.Request.URL; rawURL != nil {
		parsedURL, err := parseAndBuildURL(*raw.Request.URL, raw.Request.QueryParams)
		if err != nil {
			return config.Config{}, err
		}
		cfg.Request.URL = parsedURL
		fields = append(fields, config.FieldURL)
	}

	if header := raw.Request.Header; header != nil {
		httpHeader := http.Header{}
		for key, val := range header {
			httpHeader[key] = val
		}
		cfg.Request.Header = httpHeader
		fields = append(fields, config.FieldHeader)
	}

	if timeout := raw.Request.Timeout; timeout != nil {
		parsedTimeout, err := parseOptionalDuration(*timeout)
		if err != nil {
			return config.Config{}, err
		}
		cfg.Request.Timeout = parsedTimeout
		fields = append(fields, config.FieldTimeout)
	}

	if requests := raw.RunnerOptions.Requests; requests != nil {
		cfg.RunnerOptions.Requests = *requests
		fields = append(fields, config.FieldRequests)
	}

	if concurrency := raw.RunnerOptions.Concurrency; concurrency != nil {
		cfg.RunnerOptions.Concurrency = *concurrency
		fields = append(fields, config.FieldConcurrency)
	}

	if interval := raw.RunnerOptions.Interval; interval != nil {
		parsedInterval, err := parseOptionalDuration(*interval)
		if err != nil {
			return config.Config{}, err
		}
		cfg.RunnerOptions.Interval = parsedInterval
		fields = append(fields, config.FieldInterval)
	}

	if globalTimeout := raw.RunnerOptions.GlobalTimeout; globalTimeout != nil {
		parsedGlobalTimeout, err := parseOptionalDuration(*globalTimeout)
		if err != nil {
			return config.Config{}, err
		}
		cfg.RunnerOptions.GlobalTimeout = parsedGlobalTimeout
		fields = append(fields, config.FieldGlobalTimeout)
	}

	body := config.Body{Type: raw.Request.Body.Type, Content: []byte(raw.Request.Body.Content)}
	if !reflect.DeepEqual(body, config.NewBody("", "")) {
		cfg.Request.Body = body
		fields = append(fields, config.FieldBody)
	}

	return config.Default().Override(cfg, fields...), nil
}

// parseAndBuildURL parses a raw string as a *url.URL and adds any extra
// query parameters. It returns the first non-nil error occurring in the
// process.
func parseAndBuildURL(raw string, qp map[string]string) (*url.URL, error) {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return nil, err
	}

	// retrieve url query, add extra params, re-attach to url
	if qp != nil {
		q := u.Query()
		for k, v := range qp {
			q.Add(k, v)
		}
		u.RawQuery = q.Encode()
	}

	return u, nil
}

// parseOptionalDuration parses the raw string as a time.Duration
// and returns the parsed value or a non-nil error.
// Contrary to time.ParseDuration, it does not return an error
// if raw == "".
func parseOptionalDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}
