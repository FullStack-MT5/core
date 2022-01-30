package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Report struct {
	Records []Record `json:"records"`
	Length  int      `json:"length"`
	Success int      `json:"success"`
	Fail    int      `json:"fail"`
}

// Collects pulls the records from the given channel as soon as
// they are available and consumes them to build the report.
// When all the records have been collected and the channel is
// closed, returns the report.
// Blocks until the records channel is empty.
func (r *Requester) Collect() Report {
	rep := Report{}

	for rec := range r.Records {
		if rec.Error != nil {
			rep.Fail++
		} else {
			rep.Records = append(rep.Records, rec)
		}
		rep.Length++
	}
	rep.Length = len(rep.Records)

	return rep
}

// Send sends the report to url. Returns a non-nil error if any
// occurs during the process.
func (r *Requester) Send(url string, report Report) error {
	body := bytes.Buffer{}
	if err := json.NewEncoder(&body).Encode(report); err != nil {
		return fmt.Errorf("error sending the report: %s", err)
	}

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return fmt.Errorf("error sending the report: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending the report: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("error sending the report: %s", resp.Status)
	}

	return nil
}

// CollectAndSend calls Collect and then Send in a single
// invocation. It's useful for simple usecases where the
// caller don't need to known about the Report.
func (r *Requester) CollectAndSend(url string) error {
	report := r.Collect()

	if err := r.Send(url, report); err != nil {
		return err
	}
	return nil
}