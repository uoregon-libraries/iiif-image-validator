package validator

import (
	cryptorand "crypto/rand"
	"fmt"
	"image"
	"math/rand"
)

// Size is an image width/height pair, printed Python-tuple style so failure
// output matches the original validator.
type Size struct{ W, H int }

func (s Size) String() string { return fmt.Sprintf("(%d, %d)", s.W, s.H) }

func imgSize(img image.Image) Size {
	b := img.Bounds()
	return Size{b.Dx(), b.Dy()}
}

// checkGridSquares samples n random grid squares from img (scaled so one
// reference square is sqs pixels) and verifies their colors, passing if at
// least need of them match.
func checkGridSquares(r *Result, img image.Image, sqs, box, n, need int) error {
	match := 0
	for i := 0; i < n; i++ {
		x, y := rand.Intn(10), rand.Intn(10)
		xi, yi := x*sqs+13, y*sqs+13
		sqr := crop(img, xi, yi, xi+box, yi+box)
		if r.TestSquare(sqr, x, y) {
			match++
		}
	}
	if match >= need {
		return nil
	}
	return r.fail("color", 1, 0, "")
}

// countPixels counts the pixels of img satisfying pred.
func countPixels(img image.Image, pred func(r, g, b uint8) bool) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			pr, pg, pb, _ := img.At(x, y).RGBA()
			if pred(uint8(pr>>8), uint8(pg>>8), uint8(pb>>8)) {
				n++
			}
		}
	}
	return n
}

// randomUUID produces a v4-style UUID string for not-found identifier tests.
func randomUUID() string {
	b := make([]byte, 16)
	cryptorand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// isJSONInt reports whether a decoded JSON value is an integral number.
func isJSONInt(v any) bool {
	f, ok := v.(float64)
	return ok && f == float64(int64(f))
}
