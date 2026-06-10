package iiiftest

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
)

// The md5 of html/67352ccc-d1b0-11e1-89ae-279075081939.jp2 in the official
// IIIF image-validator repository.
const officialJP2MD5 = "f022a5ddf0ef3ecbae5f5e5c9e9e45d3"

func TestEmbeddedJP2IsOfficial(t *testing.T) {
	if got := fmt.Sprintf("%x", md5.Sum(referenceJP2)); got != officialJP2MD5 {
		t.Errorf("embedded reference.jp2 md5 = %s, want %s (official CC0 reference image)", got, officialJP2MD5)
	}
}

func TestServerServesRealJP2(t *testing.T) {
	srv := httptest.NewServer(NewServer("2.0", ""))
	defer srv.Close()

	get := func(path string) []byte {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	// untransformed request gets the real, full JP2
	data := get("/" + Identifier + "/full/full/0/default.jp2")
	if got := fmt.Sprintf("%x", md5.Sum(data)); got != officialJP2MD5 {
		t.Errorf("full jp2 md5 = %s, want official reference image", got)
	}

	// transformed requests fall back to the magic-bytes stub
	data = get("/" + Identifier + "/full/500,500/0/default.jp2")
	if len(data) != 12 {
		t.Errorf("transformed jp2 should be the 12-byte stub, got %d bytes", len(data))
	}
}
