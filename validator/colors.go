package validator

import (
	"fmt"
	"image"
	"math/rand"
)

// RGB is an 8-bit color triple.
type RGB struct{ R, G, B uint8 }

// ColorInfo is the 10x10 grid of expected square colors in the reference
// test image, indexed [x][y].
var ColorInfo = [10][10]RGB{
	{{61, 170, 126}, {61, 107, 178}, {82, 85, 234}, {164, 122, 110}, {129, 226, 88}, {91, 37, 121}, {138, 128, 42}, {6, 85, 234}, {121, 109, 204}, {65, 246, 84}},
	{{195, 133, 120}, {171, 43, 102}, {118, 45, 130}, {242, 105, 171}, {5, 85, 105}, {113, 58, 41}, {223, 69, 3}, {45, 79, 140}, {35, 117, 248}, {121, 156, 184}},
	{{168, 92, 163}, {28, 91, 143}, {86, 41, 173}, {111, 230, 29}, {174, 189, 7}, {18, 139, 88}, {93, 168, 128}, {35, 2, 14}, {204, 105, 137}, {18, 86, 128}},
	{{107, 55, 178}, {251, 40, 184}, {47, 36, 139}, {2, 127, 170}, {224, 12, 114}, {133, 67, 108}, {239, 174, 209}, {85, 29, 156}, {8, 55, 188}, {240, 125, 7}},
	{{112, 167, 30}, {166, 63, 161}, {232, 227, 23}, {74, 80, 135}, {79, 97, 47}, {145, 160, 80}, {45, 160, 79}, {12, 54, 215}, {203, 83, 70}, {78, 28, 46}},
	{{102, 193, 63}, {225, 55, 91}, {107, 194, 147}, {167, 24, 95}, {249, 214, 96}, {167, 34, 136}, {53, 254, 209}, {172, 222, 21}, {153, 77, 51}, {137, 39, 183}},
	{{159, 182, 192}, {128, 252, 173}, {148, 162, 90}, {192, 165, 115}, {154, 102, 2}, {107, 237, 62}, {111, 236, 219}, {129, 113, 172}, {239, 204, 166}, {60, 96, 37}},
	{{72, 172, 227}, {119, 51, 100}, {209, 85, 165}, {87, 172, 159}, {188, 42, 162}, {99, 3, 54}, {7, 42, 37}, {105, 155, 100}, {38, 220, 240}, {98, 46, 2}},
	{{18, 223, 145}, {189, 121, 17}, {88, 3, 210}, {181, 16, 43}, {189, 39, 244}, {123, 147, 116}, {246, 148, 214}, {223, 177, 199}, {77, 18, 136}, {235, 36, 21}},
	{{146, 137, 176}, {84, 248, 55}, {61, 144, 79}, {110, 251, 49}, {43, 105, 132}, {165, 131, 55}, {60, 23, 225}, {147, 197, 226}, {80, 67, 104}, {161, 119, 182}},
}

// dominantColor returns the most frequent color in img, reduced to 8-bit RGB.
func dominantColor(img image.Image) RGB {
	counts := map[RGB]int{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			c := RGB{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}
			counts[c]++
		}
	}
	var best RGB
	bestN := -1
	for c, n := range counts {
		if n > bestN {
			best, bestN = c, n
		}
	}
	return best
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// TestSquare checks that the dominant color of img matches grid square
// (x, y) of the reference image within the same tolerance as the Python
// validator. The outcome is recorded in the result's check list.
func (r *Result) TestSquare(img image.Image, x, y int) bool {
	truth := ColorInfo[x][y]
	got := dominantColor(img)
	ok := absDiff(got.R, truth.R) < 6 && absDiff(got.G, truth.G) < 6 && absDiff(got.B, truth.B) < 6
	r.Checks = append(r.Checks, fmt.Sprintf("%d,%d:%v", x, y, pyBool(ok)))
	return ok
}

// pyBool renders booleans the way Python str() does so check lists look the
// same as the original validator's output.
func pyBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// crop returns the subimage of img within rect (translated to img's bounds
// origin), for images whose type supports SubImage; otherwise it copies.
func crop(img image.Image, x0, y0, x1, y1 int) image.Image {
	rect := image.Rect(x0, y0, x1, y1).Add(img.Bounds().Min)
	type subImager interface {
		SubImage(image.Rectangle) image.Image
	}
	if si, ok := img.(subImager); ok {
		return si.SubImage(rect)
	}
	out := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			out.Set(x-x0, y-y0, img.At(img.Bounds().Min.X+x, img.Bounds().Min.Y+y))
		}
	}
	return out
}

// randomString generates length characters in the range '0'..'z', avoiding
// path-segment terminators (?, #, /), like the Python make_randomstring.
func randomString(length int) string {
	out := make([]byte, length)
	for i := range out {
		c := byte(rand.Intn(122-48+1) + 48)
		if c == '?' || c == '#' || c == '/' {
			c = '$'
		}
		out[i] = c
	}
	return string(out)
}
