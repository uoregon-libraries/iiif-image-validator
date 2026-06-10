// Package validator implements the IIIF Image API validation tests,
// a Go port of the official Python iiif-validator.
package validator

import "fmt"

// Error is a validation failure, mirroring the Python ValidatorError. Got and
// Expected are loosely typed because tests compare statuses, strings, sizes
// and colors alike.
type Error struct {
	Type     string
	Got      any
	Expected any
	Message  string
	Warning  bool

	// State captured from the client at the time of failure
	URL     string
	Status  int
	Headers map[string][]string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Expected %v for %s; Got: %v (%s)", e.Expected, e.Type, e.Got, e.Message)
	}
	return fmt.Sprintf("Expected %v for %s; Got: %v", e.Expected, e.Type, e.Got)
}
