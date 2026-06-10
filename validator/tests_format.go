package validator

import (
	"bytes"
	"fmt"
	"strings"
)

// formatTest builds a test that requests format ext and verifies the
// response decodes as pilFormat.
func formatTest(name, label string, level int, levels map[string]int, versions []string, ext, pilFormat string) *Test {
	return &Test{
		Name: name, Label: label, Level: level, Levels: levels, Category: 6,
		Versions: versions,
		Run: func(r *Result) error {
			_, format, err := r.Client.GetImage(Params{Format: ext})
			if err != nil {
				if serr := r.checkStatus(200, "Failed to retrieve "+pilFormat+" image."); serr != nil {
					return serr
				}
				return r.fail("quality", err.Error(), pilFormat, "Failed to decode "+pilFormat+" image.")
			}
			return r.check("quality", format, pilFormat, "")
		},
	}
}

// magicTest builds a test for formats Go cannot decode, verifying status 200
// and the response's magic bytes.
func magicTest(name, label string, level int, levels map[string]int, versions []string, ext, expected string, matches func([]byte) bool) *Test {
	return &Test{
		Name: name, Label: label, Level: level, Levels: levels, Category: 6,
		Versions: versions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Format: ext})
			data, err := r.Client.Fetch(url)
			if err != nil {
				return r.fail("format", err.Error(), url, "Failed to retrieve "+ext+": "+err.Error())
			}
			if r.Client.LastStatus != 200 {
				return r.fail("format", fmt.Sprintf("http response code: %d", r.Client.LastStatus), url,
					fmt.Sprintf("Failed to retrieve %s, got response code %d", ext, r.Client.LastStatus))
			}
			if !matches(data) {
				return r.fail("format", "unknown", expected, "")
			}
			r.Checks = append(r.Checks, "format")
			return nil
		},
	}
}

func init() {
	register(formatTest("format_jpg", "JPG format",
		1, map[string]int{"3.0": 0, "2.0": 0, "1.0": 1, "1.1": 1},
		AllVersions, "jpg", "JPEG"))
	register(formatTest("format_png", "PNG format", 2, nil, AllVersions, "png", "PNG"))
	register(formatTest("format_gif", "GIF format", 3, nil, AllVersions, "gif", "GIF"))
	register(formatTest("format_tif", "TIFF format", 3, nil, AllVersions, "tif", "TIFF"))

	register(magicTest("format_webp", "WebP format", 3, nil, []string{"2.0", "3.0"},
		"webp", "WEBP", func(data []byte) bool {
			return len(data) >= 12 && string(data[8:12]) == "WEBP"
		}))
	register(magicTest("format_jp2", "JPEG2000 format",
		3, map[string]int{"3.0": 3, "2.0": 3, "1.0": 2, "1.1": 3},
		AllVersions, "jp2", "JPEG 2000", func(data []byte) bool {
			jp2Sig := []byte{0x00, 0x00, 0x00, 0x0c, 0x6a, 0x50, 0x20, 0x20, 0x0d, 0x0a, 0x87, 0x0a}
			j2kSig := []byte{0xff, 0x4f, 0xff, 0x51}
			return bytes.HasPrefix(data, jp2Sig) || bytes.HasPrefix(data, j2kSig)
		}))
	register(magicTest("format_pdf", "PDF format", 3, nil, AllVersions,
		"pdf", "PDF", func(data []byte) bool {
			return bytes.HasPrefix(data, []byte("%PDF"))
		}))

	register(&Test{
		Name: "format_conneg", Label: "Negotiated format", Level: 1, Category: 7,
		Versions: []string{"1.0", "1.1"},
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{NoFormat: true})
			if _, err := r.Client.FetchWithHeaders(url, map[string]string{"Accept": "image/png;q=1.0"}); err != nil {
				return r.fail("format", err.Error(), "image/png", "Failed to negotiate png: "+err.Error())
			}
			ct := r.Client.LastHeaders.Get("Content-Type")
			if i := strings.IndexByte(ct, ';'); i > -1 {
				ct = strings.TrimSpace(ct[:i])
			}
			return r.check("format", ct, "image/png", "")
		},
	})

	register(&Test{
		Name: "format_error_random", Label: "Random format gives 400", Level: 1, Category: 6,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Format: randomString(3)})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 400, "Failed to get random format from url: "+url+".")
			}
			return r.check("status", r.Client.LastStatus, []int{400, 415, 503}, "")
		},
	})
}
