package validator

import (
	"fmt"
	"slices"
	"sort"
)

// Versions supported by the validator.
var AllVersions = []string{"1.0", "1.1", "2.0", "3.0"}

// Test is one validation test. Name matches the Python module name (e.g.
// "size_wh") so --test selections are compatible with the original CLI.
type Test struct {
	Name     string
	Label    string
	Level    int            // compliance level the test belongs to
	Levels   map[string]int // optional per-version override of Level
	Category int
	Versions []string
	Run      func(r *Result) error
}

// LevelFor returns the compliance level of this test under the given API
// version (or the max across versions when version is empty, like Python's
// make_info).
func (t *Test) LevelFor(version string) int {
	if len(t.Levels) == 0 {
		return t.Level
	}
	if version != "" {
		if l, ok := t.Levels[version]; ok {
			return l
		}
		return t.Level
	}
	max := 0
	for _, l := range t.Levels {
		if l > max {
			max = l
		}
	}
	return max
}

// AppliesTo reports whether the test is defined for the given API version.
func (t *Test) AppliesTo(version string) bool {
	if version == "" {
		return true
	}
	return slices.Contains(t.Versions, version)
}

var registry = map[string]*Test{}

func register(t *Test) {
	if _, dup := registry[t.Name]; dup {
		panic("duplicate test name: " + t.Name)
	}
	registry[t.Name] = t
}

// Tests returns all tests applicable to version (all tests if version is
// empty), sorted by name.
func Tests(version string) []*Test {
	var out []*Test
	for _, t := range registry {
		if t.AppliesTo(version) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup finds a test by name.
func Lookup(name string) (*Test, bool) {
	t, ok := registry[name]
	return t, ok
}

// RunTest executes one named test against a fresh client and returns its
// result. Tests signal failure via *Error; any other error is wrapped as an
// internal-error.
func RunTest(name string, c *Client) (*Result, error) {
	t, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no such test: %s", name)
	}
	r := &Result{Name: name, Label: t.Label, Client: c}
	err := t.Run(r)
	if err != nil {
		if verr, ok := err.(*Error); ok {
			r.Err = verr
		} else {
			r.Err = r.fail("internal-error", err.Error(), "no error", err.Error())
		}
	}
	return r, nil
}
