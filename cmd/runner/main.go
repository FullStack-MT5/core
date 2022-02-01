package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/benchttp/runner/config"
	configfile "github.com/benchttp/runner/config/file"
	"github.com/benchttp/runner/requester"
)

const (
	serverURL = "http://localhost:9000/report"
)

var (
	configFile    string
	uri           string
	concurrency   int           // Number of connections to run concurrently
	requests      int           // Number of requests to run, use duration as exit condition if omitted.
	timeout       time.Duration // Timeout for each http request
	globalTimeout time.Duration // Duration of test
)

var defaultConfigFiles = []string{
	"./.benchttp.yml",
	"./.benchttp.yaml",
	"./.benchttp.json",
}

func parseArgs() {
	flag.StringVar(&configFile, "configFile", configfile.Find(defaultConfigFiles), "Config file path")
	flag.StringVar(&uri, "url", "", "Target URL to request")
	flag.IntVar(&concurrency, "concurrency", 0, "Number of connections to run concurrently")
	flag.IntVar(&requests, "requests", 0, "Number of requests to run, use duration as exit condition if omitted")
	flag.DurationVar(&timeout, "timeout", 0, "Timeout for each http request")
	flag.DurationVar(&globalTimeout, "globalTimeout", 0, "Duration of test")
	flag.Parse()
}

func main() {
	parseArgs()

	cfg := parseConfig()
	fmt.Println(cfg)

	if err := requester.New(cfg).RunAndReport(serverURL); err != nil {
		log.Fatal(err)
	}
}

// parseConfig returns a config.Config initialized with config file
// options if found, overridden with CLI options.
func parseConfig() config.Config {
	fileCfg, err := configfile.Parse(configFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		// config file is not mandatory, other errors are critical
		log.Fatal(err)
	}

	cliCfg := config.New(uri, requests, concurrency, timeout, globalTimeout)

	return fileCfg.Override(cliCfg, flagNames()...)
}

// flagNames returns a slice of all flags set.
func flagNames() []string {
	var fields []string
	flag.CommandLine.Visit(func(f *flag.Flag) {
		fields = append(fields, f.Name)
	})
	return fields
}
