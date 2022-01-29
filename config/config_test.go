package config_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/benchttp/runner/config"
)

func TestMerge(t *testing.T) {
	t.Run("do not override with zero values", func(t *testing.T) {
		cfgBase := newConfig()
		cfgZero := config.Config{}

		if got := config.Merge(cfgBase, cfgZero); !reflect.DeepEqual(got, cfgBase) {
			t.Errorf("overrode with zero values: exp %#v\ngot %#v", cfgBase, got)
		}
	})

	t.Run("override with non-zero values", func(t *testing.T) {
		cfgBase := newConfig()
		cfgOver := config.Config{
			Request: config.Request{
				Method: "POST",
				URL: &url.URL{
					Host: "example",
				},
				Timeout: 2 * time.Second,
			},
			RunnerOptions: config.RunnerOptions{
				Requests:      2,
				Concurrency:   2,
				GlobalTimeout: 2 * time.Second,
			},
		}

		if got := config.Merge(cfgBase, cfgOver); !reflect.DeepEqual(got, cfgOver) {
			t.Errorf(
				"did not override with non-zero values: exp %#v\ngot %#v",
				cfgOver, got,
			)
		}
	})

	t.Run("override with non-zero values selectively", func(t *testing.T) {
		cfgBase := newConfig()
		cfgOver := config.Config{}
		cfgOver.Request.Method = "POST"
		cfgOver.RunnerOptions.Concurrency = 10

		exp := config.Config{
			Request: config.Request{
				Method:  cfgOver.Request.Method,
				URL:     cfgBase.Request.URL,
				Timeout: cfgBase.Request.Timeout,
			},
			RunnerOptions: config.RunnerOptions{
				Requests:      cfgBase.RunnerOptions.Requests,
				Concurrency:   cfgOver.RunnerOptions.Concurrency,
				GlobalTimeout: cfgBase.RunnerOptions.GlobalTimeout,
			},
		}

		if got := config.Merge(cfgBase, cfgOver); got != exp {
			t.Errorf(
				"did not selectively override with non-zero values: exp %#v\ngot %#v",
				exp, got,
			)
		}
	})
}

func newConfig() config.Config {
	return config.Config{
		Request: config.Request{
			Method: "GET",
			URL: &url.URL{
				Host:     "localhost",
				RawQuery: "delay=200ms",
			},
			Timeout: 1 * time.Second,
		},
		RunnerOptions: config.RunnerOptions{
			Requests:      1,
			Concurrency:   1,
			GlobalTimeout: 1 * time.Second,
		},
	}
}