package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/benchttp/runner/config"
	"github.com/benchttp/runner/internal/configfile"
	"github.com/benchttp/runner/internal/configflags"
	"github.com/benchttp/runner/output"
	"github.com/benchttp/runner/requester"
)

var defaultConfigFiles = []string{
	"./.benchttp.yml",
	"./.benchttp.yaml",
	"./.benchttp.json",
}

// cmdRun handles subcommand "benchttp run [options]".
type cmdRun struct {
	flagset *flag.FlagSet

	// defaultConfigFiles is a slice of default config files to look up for
	// by order of priority if none is provided via the -configFile flag.
	defaultConfigFiles []string

	// configFile is the parsed value for flag -configFile
	configFile string

	// config is the runner config resulting from parsing CLI flags.
	config config.Global
}

// ensure cmdRun implements command
var _ command = (*cmdRun)(nil)

// execute runs the benchttp runner: it parses CLI flags, loads config
// from config file and parsed flags, then runs the benchmark and outputs
// it according to the config.
func (cmd cmdRun) execute(args []string) error {
	fieldsSet := cmd.parseArgs(args)

	cfg, err := cmd.makeConfig(fieldsSet)
	if err != nil {
		return err
	}

	req, err := cfg.Request.Value()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	go cmd.listenOSInterrupt(cancel)

	rep, err := requester.New(cmd.requesterConfig(cfg)).Run(ctx, req)
	if err != nil {
		if errors.Is(err, requester.ErrCanceled) {
			if err := cmd.handleRunInterrupt(); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return output.New(rep, cfg).Export()
}

// parseArgs parses input args as config fields and returns
// a slice of fields that were set by the user.
func (cmd *cmdRun) parseArgs(args []string) []string {
	// first arg is subcommand "run"
	if len(args) <= 1 {
		return []string{}
	}

	// config file path
	cmd.flagset.StringVar(&cmd.configFile,
		"configFile",
		configfile.Find(cmd.defaultConfigFiles),
		"Config file path",
	)

	// cli config
	configflags.Set(cmd.flagset, &cmd.config)

	cmd.flagset.Parse(args[1:]) //nolint:errcheck // never occurs due to flag.ExitOnError

	return configflags.Which(cmd.flagset)
}

// makeConfig returns a config.Config initialized with config file
// options if found, overridden with CLI options listed in fields
// slice param.
func (cmd cmdRun) makeConfig(fields []string) (cfg config.Global, err error) {
	fileConfig, err := configfile.Parse(cmd.configFile)
	if err != nil && !errors.Is(err, configfile.ErrFileNotFound) {
		// config file is not mandatory, other errors are critical
		return
	}

	mergedConfig := fileConfig.Override(cmd.config, fields...)

	return mergedConfig, mergedConfig.Validate()
}

// requesterConfig returns a requester.Config generated from cfg.
func (cmd cmdRun) requesterConfig(cfg config.Global) requester.Config {
	return requester.Config{
		Requests:       cfg.Runner.Requests,
		Concurrency:    cfg.Runner.Concurrency,
		Interval:       cfg.Runner.Interval,
		RequestTimeout: cfg.Runner.RequestTimeout,
		GlobalTimeout:  cfg.Runner.GlobalTimeout,
		Silent:         cfg.Output.Silent,
	}
}

// listenOSInterrupt listens for OS interrupt signals and calls callback.
// It should be called in a separate goroutine from main as it blocks
// the execution until the OS interrupt signal is received.
func (cmd cmdRun) listenOSInterrupt(callback func()) {
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt)
	<-sigC
	callback()
}

// handleRunInterrupt handles the case when the runner is interrupted.
func (cmd cmdRun) handleRunInterrupt() error {
	reader := bufio.NewReader(os.Stdin)
	// TODO: list output strategies
	// TODO: do not prompt if strategy is stdout only
	// TODO: add config option "output.generateOnCancel" and remove prompt?
	fmt.Printf("\nBenchmark interrupted, generate output anyway? (yes/no): ")
	line, _, err := reader.ReadLine()
	if err != nil {
		return err
	}
	if string(line) != "yes" {
		return errors.New("benchmark interrupted without output")
	}
	return nil
}