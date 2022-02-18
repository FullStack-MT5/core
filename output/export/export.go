package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

var (
	// ErrJSONMarshal reports an error marshaling JSON.
	ErrJSONMarshal = errors.New("export: error marshaling JSON")
	// ErrFileCreate reports an error creating a file.
	ErrFileCreate = errors.New("export: error creating file")
	// ErrFileWrite reports an error writing a file.
	ErrFileWrite = errors.New("export: error writing file")
	// ErrHTTPRequest reports an HTTP request error, creating or sending it.
	ErrHTTPRequest = errors.New("export: request error")
	// ErrHTTPResponse reports an HTTP response error, such as bad status code.
	ErrHTTPResponse = errors.New("export: server response error")
)

// Interface gathers the necessary methods to use any function exposed
// in this package.
type Interface interface {
	fmt.Stringer
	HTTPRequester
}

// HTTPRequester interface expects a methods HTTPRequest returning
// a *http.Request to be sent in func HTTP.
type HTTPRequester interface {
	HTTPRequest() (*http.Request, error)
}

// Stdout writes src to os.Stdout.
func Stdout(src fmt.Stringer) {
	fmt.Println(src)
}

// JSONFile marshals src to JSON and write the result to a file
// with the given filename.
func JSONFile(filename string, src interface{}) error {
	b, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: %s", ErrJSONMarshal, err)
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFileCreate, err)
	}

	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("%w: %s", ErrFileWrite, err)
	}

	return nil
}

// HTTP sends the HTTP Request created by src and returns the first error
// occurring in the process. The error value can be:
// 	- ErrHTTPRequest if it fails to create or send the request
// 	- ErrHTTPResponse if the response returned a bad status code
// 	- nil otherwise.
func HTTP(src HTTPRequester) error {
	req, err := src.HTTPRequest()
	if err != nil {
		return fmt.Errorf("%w: creation: %s", ErrHTTPRequest, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: send: %s", ErrHTTPRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: response code %d", ErrHTTPResponse, resp.StatusCode)
	}

	return nil
}
