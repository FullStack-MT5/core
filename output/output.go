package output

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/benchttp/runner/ansi"
	"github.com/benchttp/runner/config"
	"github.com/benchttp/runner/output/export"
	"github.com/benchttp/runner/requester"
)

// Output represent a benchmark result as exported by the runner.
type Output struct {
	Report   requester.Report
	Metadata struct {
		Config     config.Global
		FinishedAt time.Time
	}

	log func(v ...interface{})
}

// New returns an Output initialized with rep and cfg.
func New(rep requester.Report, cfg config.Global) *Output {
	outputLogger := newLogger(cfg.Output.Silent)
	return &Output{
		Report: rep,
		Metadata: struct {
			Config     config.Global
			FinishedAt time.Time
		}{
			Config:     cfg,
			FinishedAt: time.Now(),
		},

		log: outputLogger.Println,
	}
}

// newLogger returns the logger to be used by Output.
func newLogger(silent bool) *log.Logger {
	var writer io.Writer = os.Stdout
	if silent {
		writer = nopWriter{}
	}
	return log.New(writer, ansi.Bold("→ "), 0)
}

// Export exports an Output using the Strategies set in the attached
// config.Global. If any error occurs for a given Strategy, it does not
// block the other exports and returns an ExportError listing the errors.
func (o Output) Export() error {
	var ok bool
	var errs []error

	s := exportStrategy(o.Metadata.Config.Output.Out)
	if s.is(Stdout) {
		o.log(ansi.Bold("Summary"))
		export.Stdout(o)
		ok = true
	}
	if s.is(JSONFile) {
		filename := genFilename()
		if err := export.JSONFile(filename, o); err != nil {
			errs = append(errs, err)
		} else {
			o.log(ansi.Bold("JSON generated"))
			fmt.Println(filename) // always print output filename
		}
		ok = true
	}
	if s.is(Benchttp) {
		if err := export.HTTP(o); err != nil {
			errs = append(errs, err)
		} else {
			o.log(ansi.Bold("Report sent to Benchttp"))
		}
		ok = true
	}

	if !ok {
		return ErrInvalidStrategy
	}
	if len(errs) != 0 {
		return &ExportError{Errors: errs}
	}
	return nil
}

// export.Interface implementation

var _ export.Interface = (*Output)(nil)

// String returns a default summary of an Output as a string.
func (o Output) String() string {
	if o.Metadata.Config.Output.Template != "" {
		return o.applyTemplate()
	}

	line := func(name string, value interface{}) string {
		const template = "%-18s %v\n"
		return fmt.Sprintf(template, name, value)
	}

	msString := func(d time.Duration) string {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	formatRequests := func(n, max int) string {
		maxString := strconv.Itoa(max)
		if maxString == "-1" {
			maxString = "∞"
		}
		return fmt.Sprintf("%d/%s", n, maxString)
	}

	var (
		b strings.Builder

		cfg            = o.Metadata.Config
		rep            = o.Report
		min, max, mean = rep.Stats()
	)

	b.WriteString(line("Endpoint", cfg.Request.URL))
	b.WriteString(line("Requests", formatRequests(rep.Length, cfg.Runner.Requests)))
	b.WriteString(line("Errors", rep.Fail))
	b.WriteString(line("Min response time", msString(min)))
	b.WriteString(line("Max response time", msString(max)))
	b.WriteString(line("Mean response time", msString(mean)))
	b.WriteString(line("Test duration", msString(rep.Duration)))
	return b.String()
}

func (o Output) applyTemplate() string {
	t, err := template.New("template").Parse(o.Metadata.Config.Output.Template)
	if err != nil {
		return err.Error()
	}
	var b strings.Builder
	if err := t.Execute(&b, o); err != nil {
		return err.Error()
	}
	return b.String()
}

// HTTPRequest returns the *http.Request to be sent to Benchttp server.
// The output is encoded as gob in the request body.
func (o Output) HTTPRequest() (*http.Request, error) {
	// Encode request body as gob
	b, err := encodeGob(o)
	if err != nil {
		return nil, err
	}

	// Create request
	r, err := http.NewRequest("POST", benchttpEndpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	return r, nil
}

// helpers

// encodeGob encodes the given Output as gob-encoded bytes.
func encodeGob(o Output) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(o); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// genFilename generates a JSON file name suffixed with a timestamp
// located in the working directory.
func genFilename() string {
	return fmt.Sprintf("./benchttp.report.%s.json", timestamp())
}

// timestamp returns the current time in format yy-mm-ddThh:mm:ssZhh:mm.
func timestamp() string {
	now := time.Now().UTC()
	y, m, d := now.Date()
	hh, mm, ss := now.Clock()
	return strings.ReplaceAll(
		fmt.Sprintf("%4d%2d%2d%2d%2d%2d", y, m, d, hh, mm, ss),
		" ", "0",
	)
}

type nopWriter struct{}

func (nopWriter) Write(b []byte) (int, error) { return 0, nil }
