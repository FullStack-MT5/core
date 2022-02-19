package output

import (
	"errors"
	"strings"
)

var (
	// ErrInvalidStrategy reports an unknown strategy set.
	ErrInvalidStrategy = errors.New("invalid strategy")

	errTemplateEmpty  = errors.New("empty template")
	errTemplateSyntax = errors.New("template syntax error")
)

// ExportErroris the error type returned by Output.Export.
type ExportError struct {
	Errors []error
}

// Error returns the joined errors of ExportError as a string.
func (e *ExportError) Error() string {
	const sep = "\n- "

	var b strings.Builder
	b.WriteString("output:")
	for _, err := range e.Errors {
		b.WriteString(sep)
		b.WriteString(err.Error())
	}

	return b.String()
}
