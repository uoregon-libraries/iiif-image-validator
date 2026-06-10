// Package iiiftest provides a mock IIIF Image API server for exercising the
// validator (and other IIIF clients) without a real image server. It serves
// the standard 10x10 color-grid reference image and implements
// version-appropriate behavior for IIIF 1.0, 1.1, 2.0 and 3.0.
package iiiftest

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/tiff"

	"github.com/uoregon-libraries/iiif-image-validator/validator"
)

const Identifier = "test-image"

// ReferenceImage builds the 1000x1000 grid of colored squares the validator
// expects.
func ReferenceImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	for gx := 0; gx < 10; gx++ {
		for gy := 0; gy < 10; gy++ {
			c := validator.ColorInfo[gx][gy]
			for y := gy * 100; y < (gy+1)*100; y++ {
				for x := gx * 100; x < (gx+1)*100; x++ {
					img.SetRGBA(x, y, color.RGBA{c.R, c.G, c.B, 255})
				}
			}
		}
	}
	return img
}

type Server struct {
	version string // "1.0", "1.1", "2.0", "3.0"
	prefix  string // e.g. "iiif"
	src     *image.RGBA
}

func NewServer(version, prefix string) *Server {
	return &Server{version: version, prefix: prefix, src: ReferenceImage()}
}

func (m *Server) v3() bool { return strings.HasPrefix(m.version, "3") }

func (m *Server) baseURL(r *http.Request) string {
	base := "http://" + r.Host
	if m.prefix != "" {
		base += "/" + m.prefix
	}
	return base + "/" + Identifier
}

func (m *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	path := strings.TrimPrefix(r.URL.Path, "/")
	if m.prefix != "" {
		rest, ok := strings.CutPrefix(path, m.prefix+"/")
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		path = rest
	}
	segs := strings.Split(path, "/")

	switch {
	case len(segs) == 1 && segs[0] == Identifier:
		// base URL redirects to the info document
		http.Redirect(w, r, m.baseURL(r)+"/info.json", http.StatusSeeOther)
	case len(segs) == 2 && segs[0] == Identifier && segs[1] == "info.json":
		m.serveInfoJSON(w, r)
	case len(segs) == 2 && segs[0] == Identifier && segs[1] == "info.xml":
		m.serveInfoXML(w)
	case len(segs) == 5:
		m.serveImage(w, r, segs)
	default:
		http.Error(w, "not found", 404)
	}
}

func (m *Server) serveInfoJSON(w http.ResponseWriter, r *http.Request) {
	id := m.baseURL(r)
	var info map[string]any
	switch m.version[0] {
	case '1':
		info = map[string]any{"identifier": Identifier, "width": 1000, "height": 1000}
		if m.version == "1.1" {
			info["@context"] = "http://library.stanford.edu/iiif/image-api/1.1/context.json"
			info["@id"] = id
		}
	case '2':
		info = map[string]any{
			"@context": "http://iiif.io/api/image/2/context.json",
			"@id":      id,
			"protocol": "http://iiif.io/api/image",
			"profile":  []any{"http://iiif.io/api/image/2/level2.json"},
			"width":    1000, "height": 1000,
			"sizes": []any{map[string]any{"width": 1000, "height": 1000}},
			"tiles": []any{map[string]any{"width": 512, "scaleFactors": []any{1, 2, 4}}},
		}
	default:
		info = map[string]any{
			"@context": "http://iiif.io/api/image/3/context.json",
			"id":       id,
			"type":     "ImageService3",
			"protocol": "http://iiif.io/api/image",
			"profile":  "level2",
			"width":    1000, "height": 1000,
		}
	}
	ct := "application/json"
	if strings.Contains(r.Header.Get("Accept"), "application/ld+json") {
		ct = "application/ld+json"
	}
	w.Header().Set("Content-Type", ct)
	json.NewEncoder(w).Encode(info)
}

func (m *Server) serveInfoXML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<info xmlns="http://library.stanford.edu/iiif/image-api/ns/">
  <identifier>%s</identifier>
  <width>1000</width>
  <height>1000</height>
</info>`, Identifier)
}

func (m *Server) serveImage(w http.ResponseWriter, r *http.Request, segs []string) {
	identifier, region, size, rotation, last := segs[0], segs[1], segs[2], segs[3], segs[4]
	if identifier != Identifier {
		http.Error(w, "unknown identifier", 404)
		return
	}
	quality := last
	format := ""
	if i := strings.LastIndexByte(last, '.'); i > -1 {
		quality, format = last[:i], last[i+1:]
	} else if strings.Contains(r.Header.Get("Accept"), "image/png") {
		format = "png" // 1.x content negotiation
	} else {
		format = "jpg"
	}

	img, ok := m.applyRegion(region)
	if !ok {
		http.Error(w, "bad region", 400)
		return
	}
	img, ok = m.applySize(img, size)
	if !ok {
		http.Error(w, "bad size", 400)
		return
	}
	img, ok = applyRotation(img, rotation)
	if !ok {
		http.Error(w, "bad rotation", 400)
		return
	}
	img, ok = applyQuality(img, quality)
	if !ok {
		http.Error(w, "bad quality", 400)
		return
	}

	profile := map[byte]string{
		'1': "http://library.stanford.edu/iiif/image-api/compliance.html",
		'2': "http://iiif.io/api/image/2/level2.json",
		'3': "http://iiif.io/api/image/3/level2.json",
	}[m.version[0]]
	if m.version == "1.1" {
		profile = "http://library.stanford.edu/iiif/image-api/1.1/compliance.html"
	}
	canonical := m.baseURL(r) + "/" + strings.Join(segs[1:], "/")
	w.Header().Set("Link", fmt.Sprintf(`<%s>;rel="profile", <%s>;rel="canonical"`, profile, canonical))

	switch format {
	case "jpg":
		w.Header().Set("Content-Type", "image/jpeg")
		jpeg.Encode(w, img, &jpeg.Options{Quality: 95})
	case "png":
		w.Header().Set("Content-Type", "image/png")
		png.Encode(w, img)
	case "gif":
		w.Header().Set("Content-Type", "image/gif")
		gif.Encode(w, img, nil)
	case "tif":
		w.Header().Set("Content-Type", "image/tiff")
		tiff.Encode(w, img, nil)
	case "webp":
		w.Header().Set("Content-Type", "image/webp")
		w.Write(append([]byte("RIFF\x00\x00\x00\x00WEBP"), []byte("VP8 stub")...))
	case "jp2":
		// Go has no JP2 encoder, so transforms can't be applied; serve the
		// embedded real JP2 for untransformed requests and a magic-bytes
		// stub otherwise (enough for the validator's format check).
		w.Header().Set("Content-Type", "image/jp2")
		if isUntransformed(region, size, rotation, quality) {
			w.Write(referenceJP2)
		} else {
			w.Write([]byte{0x00, 0x00, 0x00, 0x0c, 0x6a, 0x50, 0x20, 0x20, 0x0d, 0x0a, 0x87, 0x0a})
		}
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-1.4\n%stub\n"))
	default:
		http.Error(w, "bad format", 400)
	}
}

// referenceJP2 is the official CC0 reference image from the IIIF
// image-validator project, JP2-encoded; its pixels are identical to
// ReferenceImage().
//
//go:embed reference.jp2
var referenceJP2 []byte

// isUntransformed reports whether the request returns the full reference
// image unchanged.
func isUntransformed(region, size, rotation, quality string) bool {
	return (region == "full" || region == "square") &&
		(size == "full" || size == "max" || size == "^max") &&
		rotation == "0" &&
		(quality == "default" || quality == "native" || quality == "color")
}

func (m *Server) applyRegion(region string) (image.Image, bool) {
	w, h := 1000, 1000
	switch {
	case region == "full":
		return m.src, true
	case region == "square":
		return m.src, true // source is already square
	case strings.HasPrefix(region, "pct:"):
		parts := strings.Split(region[4:], ",")
		if len(parts) != 4 {
			return nil, false
		}
		var v [4]float64
		for i, p := range parts {
			f, err := strconv.ParseFloat(p, 64)
			if err != nil || f < 0 {
				return nil, false
			}
			v[i] = f
		}
		x0 := int(v[0] / 100 * float64(w))
		y0 := int(v[1] / 100 * float64(h))
		x1 := x0 + int(v[2]/100*float64(w))
		y1 := y0 + int(v[3]/100*float64(h))
		return cropRegion(m.src, x0, y0, x1, y1)
	default:
		parts := strings.Split(region, ",")
		if len(parts) != 4 {
			return nil, false
		}
		var v [4]int
		for i, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil || n < 0 {
				return nil, false
			}
			v[i] = n
		}
		return cropRegion(m.src, v[0], v[1], v[0]+v[2], v[1]+v[3])
	}
}

func cropRegion(src *image.RGBA, x0, y0, x1, y1 int) (image.Image, bool) {
	b := src.Bounds()
	x1 = min(x1, b.Max.X)
	y1 = min(y1, b.Max.Y)
	if x0 >= x1 || y0 >= y1 {
		return nil, false
	}
	return src.SubImage(image.Rect(x0, y0, x1, y1)), true
}

func (m *Server) applySize(img image.Image, size string) (image.Image, bool) {
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	upscale := false
	if strings.HasPrefix(size, "^") {
		if !m.v3() {
			return nil, false
		}
		upscale = true
		size = size[1:]
	}
	// in 3.0 plain sizes may not upscale; earlier versions may
	allowUp := upscale || !m.v3()

	fit := func(w, h int) (image.Image, bool) {
		if w <= 0 || h <= 0 || w > 5000 || h > 5000 {
			return nil, false
		}
		if !allowUp && (w > srcW || h > srcH) {
			return nil, false
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
		return dst, true
	}

	switch {
	case size == "full":
		if m.v3() {
			return nil, false // 3.0 replaced full with max
		}
		return img, true
	case size == "max":
		return img, true
	case strings.HasPrefix(size, "pct:"):
		f, err := strconv.ParseFloat(size[4:], 64)
		if err != nil || f <= 0 {
			return nil, false
		}
		return fit(int(f/100*float64(srcW)), int(f/100*float64(srcH)))
	case strings.HasPrefix(size, "!"):
		parts := strings.Split(size[1:], ",")
		if len(parts) != 2 {
			return nil, false
		}
		w, err1 := strconv.Atoi(parts[0])
		h, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, false
		}
		// fit inside the box preserving aspect; source is square so the
		// result is min(w,h) on each side
		s := min(w, h)
		return fit(s, s)
	default:
		parts := strings.Split(size, ",")
		if len(parts) != 2 {
			return nil, false
		}
		switch {
		case parts[0] == "" && parts[1] != "":
			h, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, false
			}
			return fit(h*srcW/srcH, h)
		case parts[1] == "" && parts[0] != "":
			w, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, false
			}
			return fit(w, w*srcH/srcW)
		case parts[0] != "" && parts[1] != "":
			w, err1 := strconv.Atoi(parts[0])
			h, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil {
				return nil, false
			}
			return fit(w, h)
		}
		return nil, false
	}
}

func applyRotation(img image.Image, rotation string) (image.Image, bool) {
	mirror := false
	if strings.HasPrefix(rotation, "!") {
		mirror = true
		rotation = rotation[1:]
	}
	deg, err := strconv.ParseFloat(rotation, 64)
	if err != nil || deg < 0 || deg > 360 {
		return nil, false
	}
	out := toRGBA(img)
	if mirror {
		out = flipH(out)
	}
	switch int(deg) % 360 {
	case 0:
		return out, true
	case 90:
		return rotate(out, 90), true
	case 180:
		return rotate(out, 180), true
	case 270:
		return rotate(out, 270), true
	default:
		// non-90 rotations only need to return some image; the validator
		// has no content checks for them
		return out, true
	}
}

func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok && rgba.Bounds().Min == (image.Point{}) {
		return rgba
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

func flipH(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, y, src.At(b.Dx()-1-x, y))
		}
	}
	return dst
}

// rotate turns src clockwise by 90, 180 or 270 degrees.
func rotate(src *image.RGBA, deg int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var dst *image.RGBA
	if deg == 180 {
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
	} else {
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			switch deg {
			case 90:
				dst.Set(h-1-y, x, src.At(x, y))
			case 180:
				dst.Set(w-1-x, h-1-y, src.At(x, y))
			case 270:
				dst.Set(y, w-1-x, src.At(x, y))
			}
		}
	}
	return dst
}

func applyQuality(img image.Image, quality string) (image.Image, bool) {
	switch quality {
	case "default", "native", "color":
		return img, true
	case "gray", "grey":
		return mapPixels(img, func(c color.RGBA) color.RGBA {
			l := uint8((299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000)
			return color.RGBA{l, l, l, 255}
		}), true
	case "bitonal":
		return mapPixels(img, func(c color.RGBA) color.RGBA {
			l := (299*int(c.R) + 587*int(c.G) + 114*int(c.B)) / 1000
			if l < 128 {
				return color.RGBA{0, 0, 0, 255}
			}
			return color.RGBA{255, 255, 255, 255}
		}), true
	default:
		return nil, false
	}
}

func mapPixels(img image.Image, f func(color.RGBA) color.RGBA) *image.RGBA {
	src := toRGBA(img)
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.SetRGBA(x, y, f(src.RGBAAt(x, y)))
		}
	}
	return dst
}
