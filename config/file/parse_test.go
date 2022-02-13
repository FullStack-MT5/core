package file_test

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/benchttp/runner/config"
	"github.com/benchttp/runner/config/file"
)

const (
	testdataConfigPath = "../../test/testdata/config"
	testURL            = "http://localhost:9999?fib=30&delay=200ms"
)

var supportedExt = []string{
	".yml",
	".yaml",
	".json",
}

// TestParse ensures the config file is open, read, and correctly parsed.
func TestParse(t *testing.T) {
	t.Run("return file errors early", func(t *testing.T) {
		testcases := []struct {
			label  string
			path   string
			expErr error
		}{
			{
				label:  "not found",
				path:   configPath("bad path"),
				expErr: file.ErrFileNotFound,
			},
			{
				label:  "unsupported extension",
				path:   configPath("badext.yams"),
				expErr: file.ErrFileExt,
			},
			{
				label:  "yaml invalid fields",
				path:   configPath("badfields.yml"),
				expErr: file.ErrParse,
			},
			{
				label:  "json invalid fields",
				path:   configPath("badfields.json"),
				expErr: file.ErrParse,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.label, func(t *testing.T) {
				gotCfg, gotErr := file.Parse(tc.path)

				if gotErr == nil {
					t.Fatal("exp non-nil error, got nil")
				}

				if !errors.Is(gotErr, tc.expErr) {
					t.Errorf("\nexp %v\ngot %v", tc.expErr, gotErr)
				}

				if !reflect.DeepEqual(gotCfg, config.Global{}) {
					t.Errorf("\nexp config.Config{}\ngot %v", gotCfg)
				}
			})
		}
	})

	t.Run("happy path for all extensions", func(t *testing.T) {
		for _, ext := range supportedExt {
			expCfg := newExpConfig()
			fname := path.Join(testdataConfigPath, "benchttp"+ext)

			gotCfg, err := file.Parse(fname)
			if err != nil {
				// critical error, stop the test
				t.Fatal(err)
			}

			expURL, gotURL := expCfg.Request.URL, gotCfg.Request.URL

			// compare *url.URLs separately, as they contain unpredictable values
			// they need special checks
			if !sameURL(gotURL, expURL) {
				t.Errorf("unexpected parsed URL:\nexp %v, got %v", expURL, gotURL)
			}

			// replace unpredictable values (undetermined query params order)
			restoreGotCfg := setTempValue(&gotURL.RawQuery, "replaced by test")
			restoreExpCfg := setTempValue(&expURL.RawQuery, "replaced by test")

			if !reflect.DeepEqual(gotCfg, expCfg) {
				t.Errorf("unexpected parsed config for %s file: exp %v\ngot %v", ext, expCfg, gotCfg)
			}

			restoreExpCfg()
			restoreGotCfg()
		}
	})

	t.Run("override default values", func(t *testing.T) {
		const (
			expRequests      = 0 // default is -1
			expGlobalTimeout = 42 * time.Millisecond
		)

		fname := path.Join(testdataConfigPath, "benchttp-zeros.yml")

		cfg, err := file.Parse(fname)
		if err != nil {
			t.Fatal(err)
		}

		if gotRequests := cfg.Runner.Requests; gotRequests != expRequests {
			t.Errorf("did not override Requests: exp %d, got %d", expRequests, gotRequests)
		}

		if gotGlobalTimeout := cfg.Runner.GlobalTimeout; gotGlobalTimeout != expGlobalTimeout {
			t.Errorf("did not override GlobalTimeout: exp %d, got %d", expGlobalTimeout, gotGlobalTimeout)
		}

		t.Log(cfg)
	})
}

// helpers

// newExpConfig returns the expected config.Config result after parsing
// one of the config files in testdataConfigPath.
func newExpConfig() config.Global {
	u, _ := url.ParseRequestURI(testURL)
	return config.Global{
		Request: config.Request{
			Method: "POST",
			URL:    u,
			Header: http.Header{
				"key0": []string{"val0", "val1"},
				"key1": []string{"val0"},
			},
			Body: config.NewBody("raw", `{"key0":"val0","key1":"val1"}`),
		},
		Runner: config.Runner{
			Requests:       100,
			Concurrency:    1,
			Interval:       50 * time.Millisecond,
			RequestTimeout: 2 * time.Second,
			GlobalTimeout:  60 * time.Second,
		},
		Output: config.Output{
			Out: []config.OutputStrategy{"benchttp", "json", "stdin"},
		},
	}
}

// sameURL returns true if a and b are the same *url.URL, taking into account
// the undeterministic nature of their RawQuery.
func sameURL(a, b *url.URL) bool {
	// check query params equality via Query() rather than RawQuery
	if !reflect.DeepEqual(a.Query(), b.Query()) {
		return false
	}

	// temporarily set RawQuery to a determined value
	for _, u := range []*url.URL{a, b} {
		defer setTempValue(&u.RawQuery, "replaced by test")()
	}

	// we can now rely on deep equality check
	return reflect.DeepEqual(a, b)
}

// setTempValue sets *ptr to val and returns a restore func that sets *ptr
// back to its previous value.
func setTempValue(ptr *string, val string) (restore func()) {
	previousValue := *ptr
	*ptr = val
	return func() {
		*ptr = previousValue
	}
}

func configPath(name string) string {
	return path.Join(testdataConfigPath, name)
}
