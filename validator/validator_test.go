package validator_test

import (
	"image"
	"image/color"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/uoregon-libraries/iiif-image-validator/validator"
	"github.com/uoregon-libraries/iiif-image-validator/validator/iiiftest"
)

func TestMakeURLDefaults(t *testing.T) {
	cases := []struct {
		version, prefix string
		params          validator.Params
		want            string
	}{
		{"2.0", "", validator.Params{}, "http://example.org/abc/full/full/0/default.jpg"},
		{"3.0", "", validator.Params{}, "http://example.org/abc/full/max/0/default.jpg"},
		{"1.1", "", validator.Params{}, "http://example.org/abc/full/full/0/native"},
		{"2.0", "iiif/2", validator.Params{}, "http://example.org/iiif/2/abc/full/full/0/default.jpg"},
		{"2.0", "", validator.Params{Quality: "grey"}, "http://example.org/abc/full/full/0/gray.jpg"},
		{"1.1", "", validator.Params{Quality: "grey"}, "http://example.org/abc/full/full/0/grey"},
		{"2.0", "", validator.Params{Region: "10,10,50,50", Size: "25,", Rotation: "90", Format: "png"},
			"http://example.org/abc/10,10,50,50/25,/90/default.png"},
	}
	for _, c := range cases {
		cl := validator.NewClient("abc", "example.org", c.prefix, "http", "", c.version)
		if got := cl.MakeURL(c.params); got != c.want {
			t.Errorf("MakeURL(%s, %+v) = %q, want %q", c.version, c.params, got, c.want)
		}
	}
}

func TestMakeInfoURL(t *testing.T) {
	cl := validator.NewClient("abc", "example.org", "iiif", "https", "", "2.0")
	if got, want := cl.MakeInfoURL("json"), "https://example.org/iiif/abc/info.json"; got != want {
		t.Errorf("MakeInfoURL = %q, want %q", got, want)
	}
	if got, want := cl.MakeInfoURL("xml"), "https://example.org/iiif/abc/info.xml"; got != want {
		t.Errorf("MakeInfoURL = %q, want %q", got, want)
	}
}

func TestParseLinks(t *testing.T) {
	links, err := validator.ParseLinks(`<http://iiif.io/api/image/2/level2.json>;rel="profile", ` +
		`<http://example.org/img/full/full/0/default.jpg>; rel="canonical"`)
	if err != nil {
		t.Fatalf("ParseLinks: %v", err)
	}
	if got := validator.URIForRel(links, "profile"); got != "http://iiif.io/api/image/2/level2.json" {
		t.Errorf("profile uri = %q", got)
	}
	if got := validator.URIForRel(links, "Canonical"); got != "http://example.org/img/full/full/0/default.jpg" {
		t.Errorf("canonical uri = %q", got)
	}
	if got := validator.URIForRel(links, "missing"); got != "" {
		t.Errorf("missing rel = %q, want empty", got)
	}
}

func TestTestSquare(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	c := validator.ColorInfo[3][7]
	for y := range 50 {
		for x := range 50 {
			img.SetRGBA(x, y, color.RGBA{c.R, c.G, c.B, 255})
		}
	}
	r := &validator.Result{Client: &validator.Client{}}
	if !r.TestSquare(img, 3, 7) {
		t.Error("TestSquare should match the exact reference color")
	}
	if r.TestSquare(img, 0, 0) {
		t.Error("TestSquare should not match a different square's color")
	}
	if len(r.Checks) != 2 || r.Checks[0] != "3,7:True" || r.Checks[1] != "0,0:False" {
		t.Errorf("Checks = %v", r.Checks)
	}
}

func TestRegistryComplete(t *testing.T) {
	// every Python test module must have a Go counterpart
	want := []string{
		"baseurl_redirect", "cors", "format_conneg", "format_error_random", "format_gif",
		"format_jp2", "format_jpg", "format_pdf", "format_png", "format_tif", "format_webp",
		"id_basic", "id_error_escapedslash", "id_error_random", "id_error_unescaped",
		"id_escaped", "id_squares", "info_json", "info_xml", "jsonld", "linkheader_canonical",
		"linkheader_profile", "quality_bitonal", "quality_color", "quality_error_random",
		"quality_grey", "region_error_random", "region_percent", "region_pixels",
		"region_square", "rot_error_random", "rot_full_basic", "rot_full_non90", "rot_mirror",
		"rot_mirror_180", "rot_region_basic", "rot_region_non90", "size_bwh", "size_ch",
		"size_error_random", "size_nofull", "size_noup", "size_percent", "size_region",
		"size_up", "size_wc", "size_wh",
	}
	for _, name := range want {
		if _, ok := validator.Lookup(name); !ok {
			t.Errorf("test %s not registered", name)
		}
	}
	if got := len(validator.Tests("")); got != len(want) {
		t.Errorf("registry has %d tests, want %d", got, len(want))
	}
}

// TestEndToEnd runs every applicable validator test against the mock IIIF
// server for each API version and expects them all to pass.
func TestEndToEnd(t *testing.T) {
	for _, version := range []string{"1.0", "1.1", "2.0", "3.0"} {
		t.Run(version, func(t *testing.T) {
			srv := httptest.NewServer(iiiftest.NewServer(version, "iiif"))
			defer srv.Close()
			server := strings.TrimPrefix(srv.URL, "http://")

			for _, tst := range validator.Tests(version) {
				c := validator.NewClient(iiiftest.Identifier, server, "iiif", "http", "", version)
				r, err := validator.RunTest(tst.Name, c)
				if err != nil {
					t.Fatalf("%s: %v", tst.Name, err)
				}
				if r.Err != nil {
					t.Errorf("%s FAILED: %v (urls: %v)", tst.Name, r.Err, r.URLs())
				}
			}
		})
	}
}

// TestEndToEndRun exercises the programmatic Run API with level filtering.
func TestEndToEndRun(t *testing.T) {
	srv := httptest.NewServer(iiiftest.NewServer("2.0", ""))
	defer srv.Close()

	rep := validator.Run(validator.Options{
		Server:     srv.URL, // scheme-prefixed form
		Identifier: iiiftest.Identifier,
		Version:    "2.0",
		Level:      2,
	})
	if len(rep.Results) == 0 {
		t.Fatal("no tests ran")
	}
	if rep.Failures() != 0 {
		for _, r := range rep.Results {
			if r.Err != nil {
				t.Errorf("%s: %v", r.Name, r.Err)
			}
		}
	}
	for _, r := range rep.Results {
		tst, _ := validator.Lookup(r.Name)
		if tst.LevelFor("2.0") > 2 {
			t.Errorf("%s is level %d but level cap was 2", r.Name, tst.LevelFor("2.0"))
		}
	}
}

// TestFailureDetection ensures the validator actually reports failures when
// a server misbehaves (here: against a 2.0 server while expecting 3.0).
func TestFailureDetection(t *testing.T) {
	srv := httptest.NewServer(iiiftest.NewServer("2.0", ""))
	defer srv.Close()
	server := strings.TrimPrefix(srv.URL, "http://")

	c := validator.NewClient(iiiftest.Identifier, server, "", "http", "", "3.0")
	r, err := validator.RunTest("info_json", c)
	if err != nil {
		t.Fatal(err)
	}
	if r.Err == nil {
		t.Error("info_json should fail against a 2.0-only server when validating 3.0")
	}
}
