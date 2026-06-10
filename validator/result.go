package validator

import "reflect"

// Result holds the state of one test run: the client used for requests, the
// individual checks that passed, and the failure (if any).
type Result struct {
	Name   string
	Label  string
	Client *Client
	Checks []string
	Err    *Error
}

// URLs returns every URL the test requested.
func (r *Result) URLs() []string { return r.Client.URLs }

// fail builds an *Error annotated with the client's last response state.
func (r *Result) fail(typ string, got, expected any, msg string) *Error {
	return r.failWarn(typ, got, expected, msg, false)
}

func (r *Result) failWarn(typ string, got, expected any, msg string, warning bool) *Error {
	return &Error{
		Type: typ, Got: got, Expected: expected, Message: msg, Warning: warning,
		URL: r.Client.LastURL, Status: r.Client.LastStatus, Headers: r.Client.LastHeaders,
	}
}

// check verifies got == expected (or, when expected is a slice, that got is
// one of its elements), recording typ as a passed check or returning an
// *Error. This mirrors ValidationInfo.check.
func (r *Result) check(typ string, got, expected any, msg string) error {
	return r.checkWarn(typ, got, expected, msg, false)
}

func (r *Result) checkWarn(typ string, got, expected any, msg string, warning bool) error {
	ev := reflect.ValueOf(expected)
	if ev.Kind() == reflect.Slice {
		found := false
		for i := 0; i < ev.Len(); i++ {
			if reflect.DeepEqual(got, ev.Index(i).Interface()) {
				found = true
				break
			}
		}
		if !found {
			return r.failWarn(typ, got, expected, msg, warning)
		}
	} else if !reflect.DeepEqual(got, expected) {
		return r.failWarn(typ, got, expected, msg, warning)
	}
	r.Checks = append(r.Checks, typ)
	return nil
}

// checkStatus is the common "did we get HTTP status X" check.
func (r *Result) checkStatus(expected any, msg string) error {
	return r.check("status", r.Client.LastStatus, expected, msg)
}
