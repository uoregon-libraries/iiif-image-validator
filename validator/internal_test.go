package validator

import (
	"strings"
	"testing"
)

func TestRandomString(t *testing.T) {
	for i := 0; i < 100; i++ {
		s := randomString(8)
		if len(s) != 8 {
			t.Fatalf("len = %d", len(s))
		}
		if strings.ContainsAny(s, "?#/") {
			t.Fatalf("randomString produced path-terminating char: %q", s)
		}
	}
}
