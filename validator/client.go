package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"strings"
	"time"

	// Image decoders used by image.Decode
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Client builds IIIF Image API URLs and fetches them, tracking the most
// recent response so tests can inspect status and headers (the Python
// ImageAPI object). It also accumulates the list of URLs visited.
type Client struct {
	Scheme     string
	Server     string // host[:port], no scheme
	Prefix     []string
	Identifier string
	Auth       string
	Version    string // "1.0", "1.1", "2.0", "3.0"
	Debug      bool

	HTTP *http.Client

	LastURL     string
	LastStatus  int
	LastHeaders http.Header
	URLs        []string
}

// NewClient sets up a client for one server/identifier/version combination.
// prefix may be empty or a path like "iiif/2".
func NewClient(identifier, server, prefix, scheme, auth, version string) *Client {
	var parts []string
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix != "" {
		parts = strings.Split(prefix, "/")
	}
	if scheme == "" {
		scheme = "http"
	}
	if version == "" {
		version = "2.0"
	}
	return &Client{
		Scheme:     scheme,
		Server:     strings.TrimSuffix(server, "/"),
		Prefix:     parts,
		Identifier: identifier,
		Auth:       auth,
		Version:    version,
		HTTP:       &http.Client{Timeout: 5 * time.Second},
	}
}

// browser-like headers matching the Python validator's requests
var requestHeaders = map[string]string{
	"Origin":     "http://iiif.io/",
	"Referer":    "http://iiif.io/api/image/validator",
	"User-Agent": "Mozilla/5.0 (Macintosh; U; Intel Mac OS X 10.5; en-US; rv:1.9.1b3pre) Gecko/20081130 Minefield/3.1b3pre",
}

// Fetch GETs url with browser-like headers, following redirects. Non-2xx
// statuses are not errors; the body, status and headers are recorded on the
// client either way. Only transport-level failures return an error.
func (c *Client) Fetch(url string) ([]byte, error) {
	return c.fetch(url, nil)
}

// FetchWithHeaders is Fetch with extra request headers (e.g. Accept).
func (c *Client) FetchWithHeaders(url string, hdrs map[string]string) ([]byte, error) {
	return c.fetch(url, hdrs)
}

func (c *Client) fetch(url string, extra map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range requestHeaders {
		req.Header.Set(k, v)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	c.LastHeaders = resp.Header
	c.LastStatus = resp.StatusCode
	c.LastURL = url
	c.URLs = append(c.URLs, url)
	return data, nil
}

// Params are the IIIF URL components for MakeURL. Empty fields get
// version-appropriate defaults.
type Params struct {
	Identifier string
	Region     string
	Size       string
	Rotation   string
	Quality    string
	Format     string // "" means no extension for 1.x; 2.0+ defaults to jpg
	NoFormat   bool   // suppress the format extension even on 2.0+
}

// MakeURL builds an image request URL, defaulting any unset parameters the
// same way the Python validator does.
func (c *Client) MakeURL(p Params) string {
	if p.Identifier == "" {
		p.Identifier = c.Identifier
	}
	if p.Region == "" {
		p.Region = "full"
	}
	if p.Size == "" {
		if c.Version == "3.0" {
			p.Size = "max"
		} else {
			p.Size = "full"
		}
	}
	if p.Rotation == "" {
		p.Rotation = "0"
	}
	if p.Quality == "" {
		if c.Version == "2.0" || c.Version == "3.0" {
			p.Quality = "default"
		} else {
			p.Quality = "native"
		}
	} else if p.Quality == "grey" && (c.Version == "2.0" || c.Version == "3.0") {
		p.Quality = "gray" // en-us in 2.0+
	}
	if p.Format == "" && !p.NoFormat && (c.Version == "2.0" || c.Version == "3.0") {
		p.Format = "jpg" // format is required in 2.0+
	}

	parts := append([]string{}, c.Prefix...)
	parts = append(parts, p.Identifier, p.Region, p.Size, p.Rotation, p.Quality)
	url := strings.Join(parts, "/")
	if p.Format != "" {
		url += "." + p.Format
	}
	url = fmt.Sprintf("%s://%s/%s", c.Scheme, c.Server, url)
	if c.Debug {
		fmt.Println(url)
	}
	return url
}

// MakeInfoURL builds the info document URL; format is "json" or "xml".
func (c *Client) MakeInfoURL(format string) string {
	if format == "" {
		format = "json"
	}
	parts := append([]string{}, c.Prefix...)
	parts = append(parts, c.Identifier, "info")
	return fmt.Sprintf("%s://%s/%s.%s", c.Scheme, c.Server, strings.Join(parts, "/"), format)
}

// GetInfo fetches and parses info.json; returns nil if the fetch or parse
// fails (matching the Python behavior).
func (c *Client) GetInfo() map[string]any {
	data, err := c.Fetch(c.MakeInfoURL("json"))
	if err != nil {
		return nil
	}
	var info map[string]any
	if err := json.Unmarshal(data, &info); err != nil {
		return nil
	}
	return info
}

// GetImage fetches an image URL and decodes it.
func (c *Client) GetImage(p Params) (image.Image, string, error) {
	url := c.MakeURL(p)
	data, err := c.Fetch(url)
	if err != nil {
		return nil, "", err
	}
	return DecodeImage(data)
}

// DecodeImage decodes image bytes, returning the image and its format name
// in PIL style ("JPEG", "PNG", "GIF", "TIFF", "BMP", "WEBP").
func DecodeImage(data []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	switch format {
	case "jpeg":
		format = "JPEG"
	default:
		format = strings.ToUpper(format)
	}
	return img, format, nil
}

// ParseLinks parses an HTTP Link header into uri -> param -> values,
// following the same rules as the Python parser (rel values are lowercased
// and split on spaces).
func ParseLinks(header string) (map[string]map[string][]string, error) {
	links := map[string]map[string][]string{}
	s := strings.TrimSpace(header)
	i := 0
	n := len(s)
	skipSpace := func() {
		for i < n && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
	}
	for i < n {
		skipSpace()
		if i >= n {
			break
		}
		if s[i] != '<' {
			return nil, fmt.Errorf("parsing Link header: expected < got %q", s[i])
		}
		i++
		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			return nil, fmt.Errorf("parsing Link header: unterminated URI")
		}
		uri := s[i : i+end]
		i += end + 1
		if _, ok := links[uri]; !ok {
			links[uri] = map[string][]string{}
		}
		// parameters until comma or end
		for {
			skipSpace()
			if i >= n || s[i] == ',' {
				i++ // past comma (or beyond end, loop exits)
				break
			}
			if s[i] != ';' {
				return nil, fmt.Errorf("parsing Link header: expected ; got %q", s[i])
			}
			i++
			skipSpace()
			eq := strings.IndexByte(s[i:], '=')
			if eq < 0 {
				return nil, fmt.Errorf("parsing Link header: expected = in param")
			}
			name := strings.TrimSpace(s[i : i+eq])
			i += eq + 1
			skipSpace()
			var value string
			if i < n && s[i] == '"' {
				i++
				vend := strings.IndexByte(s[i:], '"')
				if vend < 0 {
					return nil, fmt.Errorf("parsing Link header: unterminated quoted value")
				}
				value = s[i : i+vend]
				i += vend + 1
			} else {
				j := i
				for j < n && s[j] != ',' && s[j] != ';' && s[j] != ' ' {
					j++
				}
				value = s[i:j]
				i = j
			}
			if name == "rel" {
				for _, r := range strings.Fields(strings.ToLower(value)) {
					links[uri][name] = append(links[uri][name], r)
				}
			} else {
				links[uri][name] = append(links[uri][name], value)
			}
		}
	}
	return links, nil
}

// URIForRel returns the first URI in links carrying the given rel type.
func URIForRel(links map[string]map[string][]string, rel string) string {
	rel = strings.ToLower(rel)
	for uri, params := range links {
		for _, r := range params["rel"] {
			if r == rel {
				return uri
			}
		}
	}
	return ""
}
