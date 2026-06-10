// Command iiif-validate runs the IIIF Image API validation tests against a
// server from the command line, mirroring the Python iiif-validate.py tool.
// The exit code is the number of failed tests (0 on success).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/uoregon-libraries/iiif-image-validator/validator"
)

type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ",") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	var (
		identifier = flag.String("identifier", "", "identifier to run tests for (required)")
		server     = flag.String("server", "localhost:8000", "server name of IIIF service, including port if not port 80")
		prefix     = flag.String("prefix", "", "prefix of IIIF service on server")
		scheme     = flag.String("scheme", "http", "scheme (http or https)")
		auth       = flag.String("auth", "", "auth info for service")
		version    = flag.String("version", "2.0", "IIIF API version to test for (1.0, 1.1, 2.0 or 3.0)")
		level      = flag.Int("level", 1, "compliance level to test")
		verbose    = flag.Bool("verbose", false, "be verbose")
		quiet      = flag.Bool("quiet", false, "minimal output only for errors")
		jsonOut    = flag.Bool("json", false, "emit results as JSON on stdout")
		listTests  = flag.Bool("list", false, "list available tests for the chosen version and exit")
		tests      listFlag
	)
	flag.Var(&tests, "test", "run specific named test, ignores -level (repeatable)")
	// short aliases matching the Python CLI
	flag.StringVar(identifier, "i", "", "alias for -identifier")
	flag.StringVar(server, "s", "localhost:8000", "alias for -server")
	flag.StringVar(prefix, "p", "", "alias for -prefix")
	flag.StringVar(auth, "a", "", "alias for -auth")
	flag.BoolVar(verbose, "v", false, "alias for -verbose")
	flag.BoolVar(quiet, "q", false, "alias for -quiet")
	flag.Parse()

	if *listTests {
		for _, t := range validator.Tests(*version) {
			fmt.Printf("%-24s level %d  %s\n", t.Name, t.LevelFor(*version), t.Label)
		}
		return
	}

	if *identifier == "" {
		fmt.Fprintln(os.Stderr, "No identifier specified, aborting (-h for help)")
		os.Exit(99)
	}
	for _, name := range tests {
		if _, ok := validator.Lookup(name); !ok {
			fmt.Fprintf(os.Stderr, "No such test: %s\n", name)
			os.Exit(99)
		}
	}

	report := validator.Run(validator.Options{
		Server:     *server,
		Prefix:     *prefix,
		Identifier: *identifier,
		Scheme:     *scheme,
		Auth:       *auth,
		Version:    *version,
		Level:      *level,
		TestNames:  tests,
		Debug:      false,
	})

	if *jsonOut {
		printJSON(report)
	} else {
		printText(report, *verbose, *quiet)
	}
	os.Exit(report.Failures())
}

func printText(report *validator.Report, verbose, quiet bool) {
	for i, r := range report.Results {
		head := fmt.Sprintf("[%d] test %s", i+1, r.Name)
		switch {
		case r.Err != nil && r.Err.Warning:
			fmt.Printf("%s WARN\n", head)
			printErr(r)
		case r.Err != nil:
			fmt.Printf("%s FAIL\n", head)
			printErr(r)
		case !quiet:
			fmt.Printf("%s PASS\n", head)
			if verbose {
				fmt.Printf("  url: %v\n  tests: %v\n", r.URLs(), r.Checks)
			}
		}
	}
	if !quiet {
		fmt.Printf("Done (%d tests, %d failures, %d warnings)\n",
			len(report.Results), report.Failures(), report.Warnings())
	}
}

func printErr(r *validator.Result) {
	e := r.Err
	fmt.Printf("  url: %v\n  got: %v\n  expected: %v\n  type: %s\n  message: %s\n  warning: %v\n",
		r.URLs(), e.Got, e.Expected, e.Type, e.Message, e.Warning)
}

type jsonResult struct {
	Test     string   `json:"test"`
	Label    string   `json:"label"`
	Status   string   `json:"status"`
	URLs     []string `json:"url"`
	Checks   []string `json:"tests,omitempty"`
	Got      any      `json:"got,omitempty"`
	Expected any      `json:"expected,omitempty"`
	Type     string   `json:"type,omitempty"`
	Message  string   `json:"message,omitempty"`
	Warning  bool     `json:"warning,omitempty"`
}

func printJSON(report *validator.Report) {
	out := make([]jsonResult, 0, len(report.Results))
	for _, r := range report.Results {
		jr := jsonResult{Test: r.Name, Label: r.Label, Status: "success", URLs: r.URLs(), Checks: r.Checks}
		if r.Err != nil {
			jr.Status = "error"
			jr.Got = r.Err.Got
			jr.Expected = r.Err.Expected
			jr.Type = r.Err.Type
			jr.Message = r.Err.Message
			jr.Warning = r.Err.Warning
		}
		out = append(out, jr)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
