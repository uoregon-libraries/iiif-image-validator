package validator

import (
	"encoding/xml"
	"fmt"
	"strings"
)

func init() {
	register(&Test{
		Name: "info_json", Label: "Check Image Information", Level: 0, Category: 1,
		Versions: AllVersions,
		Run:      runInfoJSON,
	})

	register(&Test{
		Name: "info_xml", Label: "Check Image Information (XML)", Level: 0, Category: 1,
		Versions: []string{"1.0"},
		Run:      runInfoXML,
	})
}

func runInfoJSON(r *Result) error {
	c := r.Client
	info := c.GetInfo()
	if info == nil {
		if serr := r.checkStatus(200, fmt.Sprintf(
			"Failed to reach %s due to http status code: %d.", c.MakeInfoURL("json"), c.LastStatus)); serr != nil {
			return serr
		}
		ct := c.LastHeaders.Get("Content-Type")
		if serr := r.check("content-type",
			strings.Contains(ct, "application/json") || strings.Contains(ct, "application/ld+json"), true,
			"Content-type for the info.json needs to be either application/json or application/ld+json."); serr != nil {
			return serr
		}
		return r.fail("info.json is JSON", true, false, "")
	}

	if err := r.check("required-field: width", hasKey(info, "width"), true, ""); err != nil {
		return err
	}
	if err := r.check("required-field: height", hasKey(info, "height"), true, ""); err != nil {
		return err
	}
	if err := r.check("type-is-int: height", isJSONInt(info["height"]), true, ""); err != nil {
		return err
	}
	if err := r.check("type-is-int: width", isJSONInt(info["width"]), true, ""); err != nil {
		return err
	}

	if c.Version == "1.0" {
		return r.check("required-field: identifier", hasKey(info, "identifier"), true, "")
	}

	idField := "@id"
	if c.Version[0] == '3' {
		idField = "id"
	}
	if err := r.check("required-field: "+idField, hasKey(info, idField), true,
		"info.json is missing required field `"+idField+"`"); err != nil {
		return err
	}
	id, _ := info[idField].(string)
	if err := r.check("type-is-uri: "+idField, strings.HasPrefix(id, "http"), true, ""); err != nil {
		return err
	}
	wantID := strings.Replace(c.LastURL, "/info.json", "", 1)
	if err := r.check(idField+" is correct URI", id == wantID, true,
		fmt.Sprintf("Found: %s Expected: %s", id, wantID)); err != nil {
		return err
	}

	if err := r.check("required-field: @context", hasKey(info, "@context"), true, ""); err != nil {
		return err
	}
	switch {
	case c.Version == "1.1":
		if err := r.check("correct-context", info["@context"], []any{
			"http://library.stanford.edu/iiif/image-api/1.1/context.json",
			"http://iiif.io/api/image/1/context.json",
		}, ""); err != nil {
			return err
		}
	case c.Version[0] == '2':
		if err := r.check("correct-context", info["@context"],
			any("http://iiif.io/api/image/2/context.json"), ""); err != nil {
			return err
		}
	case c.Version[0] == '3':
		if list, ok := info["@context"].([]any); ok {
			found := false
			for _, v := range list {
				if v == "http://iiif.io/api/image/3/context.json" {
					found = true
				}
			}
			if err := r.check("correct-context", found, true,
				"@context missing version 3.0 IIIF context: http://iiif.io/api/image/3/context.json"); err != nil {
				return err
			}
		} else if err := r.check("correct-context", info["@context"],
			any("http://iiif.io/api/image/3/context.json"), ""); err != nil {
			return err
		}
	}

	if c.Version[0] < '2' {
		return nil
	}

	if err := r.check("required-field: protocol", hasKey(info, "protocol"), true, ""); err != nil {
		return err
	}
	if err := r.check("correct-protocol", info["protocol"], any("http://iiif.io/api/image"), ""); err != nil {
		return err
	}

	if err := r.check("required-field: profile", hasKey(info, "profile"), true, ""); err != nil {
		return err
	}
	if c.Version[0] == '2' {
		profs, ok := info["profile"].([]any)
		if err := r.check("is-list", ok && len(profs) > 0, true, "Profile should be a list."); err != nil {
			return err
		}
		first, _ := profs[0].(string)
		if err := r.check("profile-compliance",
			strings.HasPrefix(first, "http://iiif.io/api/image/2/level"), true, ""); err != nil {
			return err
		}
	} else {
		prof, _ := info["profile"].(string)
		if err := r.check("profile-compliance",
			prof == "level0" || prof == "level1" || prof == "level2", true,
			"Profile should be one of level0, level1 or level2. https://iiif.io/api/image/3.0/#6-compliance-level-and-profile-document"); err != nil {
			return err
		}
	}

	if sizes, present := info["sizes"]; present {
		list, ok := sizes.([]any)
		if err := r.check("is-list", ok, true, ""); err != nil {
			return err
		}
		for _, sz := range list {
			obj, ok := sz.(map[string]any)
			if err := r.check("is-object", ok, true, ""); err != nil {
				return err
			}
			if err := r.check("required-field: height", hasKey(obj, "height"), true, ""); err != nil {
				return err
			}
			if err := r.check("required-field: width", hasKey(obj, "width"), true, ""); err != nil {
				return err
			}
			if err := r.check("type-is-int: height", isJSONInt(obj["height"]), true, ""); err != nil {
				return err
			}
			if err := r.check("type-is-int: width", isJSONInt(obj["width"]), true, ""); err != nil {
				return err
			}
		}
	}

	if tiles, present := info["tiles"]; present {
		list, ok := tiles.([]any)
		if err := r.check("is-list", ok, true, ""); err != nil {
			return err
		}
		for _, t := range list {
			obj, ok := t.(map[string]any)
			if err := r.check("is-object", ok, true, ""); err != nil {
				return err
			}
			if err := r.check("required-field: scaleFactors", hasKey(obj, "scaleFactors"), true, ""); err != nil {
				return err
			}
			if err := r.check("required-field: width", hasKey(obj, "width"), true, ""); err != nil {
				return err
			}
			if err := r.check("type-is-int: width", isJSONInt(obj["width"]), true, ""); err != nil {
				return err
			}
		}
	}

	if c.Version[0] == '3' {
		if err := r.check("correct-type", info["type"], any("ImageService3"),
			"Info.json missing required type of ImageService3."); err != nil {
			return err
		}
		if err := r.checkWarn("license-renamed", hasKey(info, "license"), false,
			"license has been renamed rights in 3.0", true); err != nil {
			return err
		}
		if rights, present := info["rights"]; present {
			s, _ := rights.(string)
			if err := r.check("type-is-uri: rights", strings.HasPrefix(s, "http"), true,
				"Rights should be a single URI from Creative Commons, RightsStatements.org or URIs registered as extensions."); err != nil {
				return err
			}
		}
		for _, field := range []string{"extraQualities", "extraFormats", "extraFeatures"} {
			if v, present := info[field]; present {
				_, ok := v.([]any)
				if err := r.check("is-list", ok, true, field+" should be a list."); err != nil {
					return err
				}
			}
		}
		for _, name := range []string{"service", "partOf", "seeAlso"} {
			if err := checkLinkingProperty(r, name, info); err != nil {
				return err
			}
		}
		if err := r.checkWarn("attribution-missing", hasKey(info, "attribution"), false,
			"attribution has been removed in 3.0", true); err != nil {
			return err
		}
		if err := r.checkWarn("logo-missing", hasKey(info, "logo"), false,
			"logo has been removed in 3.0", true); err != nil {
			return err
		}
	}
	return nil
}

func hasKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

func checkLinkingProperty(r *Result, name string, info map[string]any) error {
	v, present := info[name]
	if !present {
		return nil
	}
	list, ok := v.([]any)
	if err := r.check("is-list", ok, true, name+" should be a list."); err != nil {
		return err
	}
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if err := r.check("is-object", ok, true,
			fmt.Sprintf("Item: %v in %s should be an object.", raw, name)); err != nil {
			return err
		}
		if name == "service" {
			if (!hasKey(item, "id") && !hasKey(item, "@id")) || (!hasKey(item, "type") && !hasKey(item, "@type")) {
				return r.fail("missing-key", "", "id, @id, type or @type",
					fmt.Sprintf("Item: %v in %s needs a id and type or @id and @type", item, name))
			}
		} else if !hasKey(item, "id") || !hasKey(item, "type") {
			return r.fail("missing-key", "", "id or type missing",
				fmt.Sprintf("Item: %v in %s needs a id and type", item, name))
		}
		if label, present := item["label"]; present {
			obj, ok := label.(map[string]any)
			if err := r.check("is-object", ok, true, "Label must be an object"); err != nil {
				return err
			}
			for lang, vals := range obj {
				_, ok := vals.([]any)
				if err := r.check("is-list", ok, true,
					"Value of Label with lng: "+lang+" should be list"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func runInfoXML(r *Result) error {
	url := r.Client.MakeInfoURL("xml")
	data, err := r.Client.Fetch(url)
	if err != nil {
		return r.fail("status", err.Error(), 200, "Failed to fetch "+url)
	}
	if serr := r.checkStatus(200, ""); serr != nil {
		return serr
	}
	ct := r.Client.LastHeaders.Get("Content-Type")
	if i := strings.IndexByte(ct, ';'); i > -1 {
		ct = strings.TrimSpace(ct[:i])
	}
	if serr := r.check("format", ct, []string{"application/xml", "text/xml"}, ""); serr != nil {
		return serr
	}

	const ns = "http://library.stanford.edu/iiif/image-api/ns/"
	var doc struct {
		XMLName    xml.Name
		Identifier []string `xml:"identifier"`
		Height     []string `xml:"height"`
		Width      []string `xml:"width"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return r.fail("format", "XML", "Unknown", "")
	}
	rootOK := 0
	if doc.XMLName.Space == ns && doc.XMLName.Local == "info" {
		rootOK = 1
	}
	if err := r.check("required-field: /info", rootOK, 1, ""); err != nil {
		return err
	}
	if err := r.check("required-field: /info/identifier", len(doc.Identifier), 1, ""); err != nil {
		return err
	}
	if err := r.check("required-field: /info/height", len(doc.Height), 1, ""); err != nil {
		return err
	}
	return r.check("required-field: /info/width", len(doc.Width), 1, "")
}
