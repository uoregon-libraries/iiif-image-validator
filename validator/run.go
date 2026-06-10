package validator

import (
	"slices"
	"strings"
)

// Options selects what to validate. Server may include a scheme prefix
// (http:// or https://) which overrides Scheme.
type Options struct {
	Server     string
	Prefix     string
	Identifier string
	Scheme     string // default "http"
	Auth       string
	Version    string // default "2.0"
	Level      int    // run tests at or below this compliance level (default 1)
	TestNames  []string
	Debug      bool
}

// Report is the outcome of a validation run.
type Report struct {
	Results []*Result
}

// Failures counts failed (non-warning) tests.
func (rep *Report) Failures() int {
	n := 0
	for _, r := range rep.Results {
		if r.Err != nil && !r.Err.Warning {
			n++
		}
	}
	return n
}

// Warnings counts tests that failed with warning-level errors.
func (rep *Report) Warnings() int {
	n := 0
	for _, r := range rep.Results {
		if r.Err != nil && r.Err.Warning {
			n++
		}
	}
	return n
}

// NewClientFromOptions builds a fresh Client for one test run.
func NewClientFromOptions(o Options) *Client {
	scheme := o.Scheme
	server := strings.TrimSpace(o.Server)
	if strings.HasPrefix(server, "https://") {
		scheme = "https"
		server = strings.TrimPrefix(server, "https://")
	} else if strings.HasPrefix(server, "http://") {
		if scheme == "" {
			scheme = "http"
		}
		server = strings.TrimPrefix(server, "http://")
	}
	c := NewClient(o.Identifier, server, o.Prefix, scheme, o.Auth, o.Version)
	c.Debug = o.Debug
	return c
}

// Run executes the selected tests (explicit TestNames, or every test at or
// below Level for the chosen Version), each against a fresh client.
func Run(o Options) *Report {
	if o.Version == "" {
		o.Version = "2.0"
	}
	rep := &Report{}
	for _, t := range Tests(o.Version) {
		if len(o.TestNames) > 0 {
			if !slices.Contains(o.TestNames, t.Name) {
				continue
			}
		} else if t.LevelFor(o.Version) > o.Level {
			continue
		}
		r, _ := RunTest(t.Name, NewClientFromOptions(o))
		rep.Results = append(rep.Results, r)
	}
	return rep
}
