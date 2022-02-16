package requester

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/benchttp/runner/config"
	"github.com/benchttp/runner/dispatcher"
)

const (
	defaultRecordsCap = 1000
)

// Requester executes the benchmark. It wraps http.Client.
type Requester struct {
	records []Record
	numErr  int
	runErr  error
	start   time.Time
	done    bool

	config config.Config
	client http.Client
	tracer *tracer

	mu sync.Mutex
}

// New returns a Requester initialized with cfg. cfg is assumed valid:
// it is the caller's responsibility to ensure cfg is valid using
// cfg.Validate.
func New(cfg config.Config) *Requester {
	recordsCap := cfg.RunnerOptions.Requests
	if recordsCap < 1 {
		recordsCap = defaultRecordsCap
	}

	tracer := newTracer()

	return &Requester{
		records: make([]Record, 0, recordsCap),
		config:  cfg,
		tracer:  tracer,
		client: http.Client{
			// Timeout includes connection time, any redirects, and reading
			// the response body.
			// We may want exclude reading the response body in our benchmark tool.
			Timeout: cfg.Request.Timeout,

			// tracer keeps track of all events of the current request.
			Transport: tracer,
		},
	}
}

// Run starts the benchmark test and pipelines the results inside a Report.
// Returns the Report when the test ended and all results have been collected.
func (r *Requester) Run() (Report, error) {
	req, err := r.config.HTTPRequest()
	if err != nil {
		return Report{}, fmt.Errorf("%w: %s", ErrRequest, err)
	}

	if err := r.ping(req); err != nil {
		return Report{}, fmt.Errorf("%w: %s", ErrConnection, err)
	}

	r.start = time.Now()

	var (
		numWorker   = r.config.RunnerOptions.Concurrency
		maxIter     = r.config.RunnerOptions.Requests
		timeout     = r.config.RunnerOptions.GlobalTimeout
		interval    = r.config.RunnerOptions.Interval
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	)

	defer cancel()

	// print state every second
	go func() {
		ticker := time.NewTicker(time.Second)
		tick := ticker.C
		for {
			if r.done {
				ticker.Stop()
				return
			}
			fmt.Print(r.state())
			<-tick
		}
	}()

	r.runErr = dispatcher.New(numWorker).Do(ctx, maxIter, r.record(req, interval))
	switch r.runErr {
	case nil, context.Canceled, context.DeadlineExceeded:
	default:
		return Report{}, r.runErr
	}

	r.done = true

	// print final state
	fmt.Println(r.state())

	return makeReport(r.config, r.records, r.numErr), nil
}

func (r *Requester) ping(req *http.Request) error {
	resp, err := r.client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	return err
}

// Record is the summary of a HTTP response. If Record.Error is non-nil,
// the HTTP call failed anywhere from making the request to decoding the
// response body, invalidating the entire response, as it is not a remote
// server error.
type Record struct {
	Time   time.Duration `json:"time"`
	Code   int           `json:"code"`
	Bytes  int           `json:"bytes"`
	Error  error         `json:"error,omitempty"`
	Events []Event       `json:"events"`
}

func (r *Requester) record(req *http.Request, interval time.Duration) func() {
	return func() {
		// It is necessary to clone the request because one request with a non-nil body cannot be used in concurrent threads
		reqClone, err := cloneRequest(req)
		if err != nil {
			r.appendRecord(Record{Error: ErrRequestBody})
			return
		}

		sent := time.Now()

		resp, err := r.client.Do(reqClone)
		if err != nil {
			r.appendRecord(Record{Error: err})
			return
		}

		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			r.appendRecord(Record{Error: err})
			return
		}

		duration := time.Since(sent)

		r.appendRecord(Record{
			Code:   resp.StatusCode,
			Time:   duration,
			Bytes:  len(body),
			Events: r.tracer.events,
		})

		fmt.Print(r.state())
		time.Sleep(interval)
	}
}

func (r *Requester) appendRecord(rec Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, rec)
	if rec.Error != nil {
		r.numErr++
	}
}

// cloneRequest fully clones a http.Request by also cloning the body via Request.GetBody
func cloneRequest(req *http.Request) (*http.Request, error) {
	reqClone := req.Clone(req.Context())
	if req.Body != nil {
		bodyClone, err := req.GetBody()
		if err != nil {
			return nil, ErrRequestBody
		}
		reqClone.Body = bodyClone
	}
	return reqClone, nil
}
