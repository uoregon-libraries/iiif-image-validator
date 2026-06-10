package validator

import (
	"net/http"
	"strings"
)

func init() {
	register(&Test{
		Name: "cors", Label: "Cross Origin Headers", Level: 1, Category: 7,
		Versions: AllVersions,
		Run: func(r *Result) error {
			r.Client.GetInfo()
			cors := r.Client.LastHeaders.Get("Access-Control-Allow-Origin")
			return r.check("CORS", cors, "*", "Failed to get correct CORS header.")
		},
	})

	register(&Test{
		Name: "jsonld", Label: "JSON-LD Media Type", Level: 1, Category: 7,
		Versions: []string{"2.0", "3.0"},
		Run: func(r *Result) error {
			url := r.Client.MakeInfoURL("json")
			if _, err := r.Client.FetchWithHeaders(url, map[string]string{"Accept": "application/ld+json"}); err != nil {
				return r.fail("json-ld", err.Error(), "application/ld+json", "Failed to fetch info.json: "+err.Error())
			}
			if r.Client.LastStatus != 200 {
				return r.checkStatus(200, "")
			}
			ct := r.Client.LastHeaders.Get("Content-Type")
			return r.check("json-ld", strings.HasPrefix(ct, "application/ld+json"), true,
				"Content-Type to start with application/ld+json")
		},
	})

	register(&Test{
		Name: "linkheader_profile", Label: "Profile Link Header", Level: 3, Category: 7,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("profile", err.Error(), "URI", "Failed to fetch url: "+url)
			}
			if serr := r.checkStatus(200, ""); serr != nil {
				return serr
			}
			lh := r.Client.LastHeaders.Get("Link")
			if lh == "" {
				return r.fail("profile", "", "URI", `Missing "link" header in response.`)
			}
			links, err := ParseLinks(lh)
			if err != nil {
				return r.fail("profile", err.Error(), "URI", "Could not parse link header.")
			}
			profile := URIForRel(links, "profile")
			if profile == "" {
				return r.fail("profile", "", "URI", "")
			}
			var want string
			switch {
			case r.Client.Version == "1.0":
				want = "http://library.stanford.edu/iiif/image-api/compliance.html"
			case r.Client.Version == "1.1":
				want = "http://library.stanford.edu/iiif/image-api/1.1/compliance.html"
			case strings.HasPrefix(r.Client.Version, "2"):
				want = "http://iiif.io/api/image/2/"
			case strings.HasPrefix(r.Client.Version, "3"):
				want = "http://iiif.io/api/image/3/"
			}
			if !strings.HasPrefix(profile, want) {
				return r.fail("profile", profile, want, "Profile link header returned unexpected link.")
			}
			r.Checks = append(r.Checks, "linkheader")
			return nil
		},
	})

	register(&Test{
		Name: "linkheader_canonical", Label: "Canonical Link Header", Level: 3, Category: 7,
		Versions: []string{"2.0", "3.0"},
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("canonical", err.Error(), "URI", "Failed to fetch url: "+url)
			}
			if serr := r.checkStatus(200, ""); serr != nil {
				return serr
			}
			lh := r.Client.LastHeaders.Get("Link")
			if lh == "" {
				return r.fail("canonical", "", "URI", `Missing "link" header in response.`)
			}
			links, err := ParseLinks(lh)
			if err != nil {
				return r.fail("canonical", err.Error(), "URI", "Could not parse link header.")
			}
			if URIForRel(links, "canonical") == "" {
				return r.fail("canonical", lh, "canonical link header", "Found link header but not canonical.")
			}
			r.Checks = append(r.Checks, "linkheader")
			return nil
		},
	})

	register(&Test{
		Name: "baseurl_redirect", Label: "Base URL Redirects", Level: 1, Category: 7,
		Versions: []string{"2.0", "3.0"},
		Run: func(r *Result) error {
			url := strings.Replace(r.Client.MakeInfoURL("json"), "/info.json", "", 1)
			// follow redirects manually so we can see whether one happened
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return r.fail("url-check", err.Error(), 301, "Failed to redirect from url: "+url+".")
			}
			resp, err := r.Client.HTTP.Do(req)
			if err != nil {
				return r.fail("url-check", err.Error(), 301, "Failed to redirect from url: "+url+".")
			}
			defer resp.Body.Close()
			r.Client.LastStatus = resp.StatusCode
			r.Client.LastHeaders = resp.Header
			r.Client.LastURL = url
			r.Client.URLs = append(r.Client.URLs, url)
			newurl := resp.Request.URL.String()
			if loc := resp.Header.Get("Location"); resp.StatusCode >= 300 && resp.StatusCode < 400 && loc != "" {
				newurl = loc
			}
			if newurl == url {
				return r.fail("redirect", newurl, url+"/info.json",
					"Failed to redirect from "+url+" to "+url+"/info.json.")
			}
			r.Checks = append(r.Checks, "redirect")
			return nil
		},
	})
}
