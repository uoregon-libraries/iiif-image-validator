package validator

import (
	"fmt"
	"math/rand"
)

func init() {
	register(&Test{
		Name: "size_wc", Label: "Size specified by w,", Level: 1, Category: 3,
		Versions: AllVersions,
		Run: func(r *Result) error {
			s := rand.Intn(750-450+1) + 450
			img, _, err := r.Client.GetImage(Params{Size: fmt.Sprintf("%d,", s)})
			if err != nil {
				return r.fail("General error", err.Error(), "No error", "Failed to check size due to: "+err.Error())
			}
			if serr := r.checkStatus(200, ""); serr != nil {
				return serr
			}
			if serr := r.check("size", imgSize(img), Size{s, s}, ""); serr != nil {
				return serr
			}
			return checkGridSquares(r, img, s/10, s/10-13, 5, 4)
		},
	})

	register(&Test{
		Name: "size_ch", Label: "Size specified by ,h", Level: 1, Category: 3,
		Versions: AllVersions,
		Run: func(r *Result) error {
			s := rand.Intn(750-450+1) + 450
			img, _, err := r.Client.GetImage(Params{Size: fmt.Sprintf(",%d", s)})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("General error", err.Error(), "No error", "Failed to check size due to: "+err.Error())
			}
			if serr := r.check("size", imgSize(img), Size{s, s}, ""); serr != nil {
				return serr
			}
			return checkGridSquares(r, img, s/10, s/10-13, 5, 4)
		},
	})

	register(&Test{
		Name: "size_wh", Label: "Size specified by w,h",
		Level: 2, Levels: map[string]int{"3.0": 1, "2.0": 2, "1.0": 2, "1.1": 2}, Category: 3,
		Versions: AllVersions,
		Run: func(r *Result) error {
			w := rand.Intn(750-350+1) + 350
			h := rand.Intn(750-350+1) + 350
			img, _, err := r.Client.GetImage(Params{Size: fmt.Sprintf("%d,%d", w, h)})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("size", err.Error(), Size{w, h}, "Failed to retrieve image at size.")
			}
			if serr := r.check("size", imgSize(img), Size{w, h}, ""); serr != nil {
				return serr
			}
			sqsw, sqsh := w/10, h/10
			match := 0
			for i := 0; i < 5; i++ {
				x, y := rand.Intn(10), rand.Intn(10)
				xi, yi := x*sqsw+13, y*sqsh+13
				sqr := crop(img, xi, yi, xi+(sqsw-13), yi+(sqsh-13))
				if r.TestSquare(sqr, x, y) {
					match++
				}
			}
			if match >= 4 {
				return nil
			}
			return r.fail("color", 1, 0, "")
		},
	})

	register(&Test{
		Name: "size_bwh", Label: "Size specified by !w,h", Level: 2, Category: 3,
		Versions: AllVersions,
		Run: func(r *Result) error {
			w := rand.Intn(750-350+1) + 350
			h := rand.Intn(750-350+1) + 350
			s := min(w, h)
			img, _, err := r.Client.GetImage(Params{Size: fmt.Sprintf("!%d,%d", w, h)})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("size", err.Error(), Size{s, s}, "Failed to retrieve image at size.")
			}
			if serr := r.check("size", imgSize(img), Size{s, s}, ""); serr != nil {
				return serr
			}
			return checkGridSquares(r, img, s/10, s/10-13, 5, 3)
		},
	})

	register(&Test{
		Name: "size_percent", Label: "Size specified by percent",
		Level: 1, Levels: map[string]int{"3.0": 2, "2.0": 1, "1.0": 1, "1.1": 1}, Category: 3,
		Versions: AllVersions,
		Run: func(r *Result) error {
			s := rand.Intn(75-45+1) + 45
			img, _, err := r.Client.GetImage(Params{Size: fmt.Sprintf("pct:%d", s)})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("General error", err.Error(), "No error", "Failed to check size due to: "+err.Error())
			}
			if serr := r.check("size", imgSize(img), Size{s * 10, s * 10}, ""); serr != nil {
				return serr
			}
			return checkGridSquares(r, img, s, s-13, 5, 4)
		},
	})

	register(&Test{
		Name: "size_region", Label: "Region at specified size", Level: 1, Category: 3,
		Versions: AllVersions,
		Run: func(r *Result) error {
			// random regions at a random size < 100 so each stays within
			// one color square of the test image
			for i := 0; i < 5; i++ {
				s := rand.Intn(90-35+1) + 35
				x, y := rand.Intn(10), rand.Intn(10)
				img, _, err := r.Client.GetImage(Params{
					Region: fmt.Sprintf("%d,%d,100,100", x*100, y*100),
					Size:   fmt.Sprintf("%d,%d", s, s),
				})
				if err != nil {
					return r.fail("General error", err.Error(), "No error", "Failed to check size due to: "+err.Error())
				}
				if imgSize(img) != (Size{s, s}) {
					return r.fail("size", imgSize(img), Size{s, s}, "")
				}
				if !r.TestSquare(img, x, y) {
					return r.fail("color", 1, ColorInfo[0][0], "")
				}
			}
			return nil
		},
	})

	register(&Test{
		Name: "size_up", Label: "Size greater than 100%", Level: 3, Category: 3,
		Versions: AllVersions,
		Run:      runSizeUp,
	})

	register(&Test{
		Name: "size_noup", Label: "Size greater than 100% should only work with the ^ notation",
		Level: 1, Category: 3, Versions: []string{"3.0"},
		Run: func(r *Result) error {
			s := rand.Intn(2000-1100+1) + 1100
			for _, sizeStr := range []string{
				fmt.Sprintf("%d,%d", s, s),
				fmt.Sprintf(",%d", s),
				fmt.Sprintf("%d,", s),
				"pct:200",
				"!2000,3000",
			} {
				_, _, err := r.Client.GetImage(Params{Size: sizeStr})
				if err != nil {
					if serr := r.checkStatus(400,
						"In version 3.0 image should only be upscaled using the ^ notation."); serr != nil {
						return serr
					}
				}
				if r.Client.LastStatus == 200 {
					return r.fail("size-upscaling", r.Client.LastStatus, "!200",
						"Retrieving upscaled image succeeded but should have failed as 3.0 requires the ^ for upscaling. Size: "+sizeStr)
				}
			}
			return nil
		},
	})

	register(&Test{
		Name: "size_nofull", Label: "3.0 replace full with max", Level: 0, Category: 3,
		Versions: []string{"3.0"},
		Run: func(r *Result) error {
			r.Client.GetImage(Params{Size: "full"})
			return r.checkWarn("size", r.Client.LastStatus != 200, true,
				"Version 3.0 has replaced the size full with max.", true)
		},
	})

	register(&Test{
		Name: "size_error_random", Label: "Random size gives 400", Level: 1, Category: 3,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Size: randomString(6)})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 400, "Failed to get random size with url "+url+".")
			}
			return r.checkStatus(400, "")
		},
	})
}

func runSizeUp(r *Result) error {
	s := rand.Intn(2000-1100+1) + 1100
	img, _, err := r.Client.GetImage(Params{Size: fmt.Sprintf(",%d", s)})
	if err == nil {
		// server upscales plain sizes (1.x/2.x behavior)
		if serr := r.check("size", imgSize(img), Size{s, s}, ""); serr != nil {
			return serr
		}
		return checkGridSquares(r, img, s/10, s/10-13, 5, 3)
	}
	if r.Client.Version[0] != '3' {
		if serr := r.checkStatus(200, "Failed to retrieve upscaled image."); serr != nil {
			return serr
		}
		return r.fail("size", err.Error(), Size{s, s}, "Failed to retrieve upscaled image.")
	}

	// 3.0: plain upscale must 400, and the ^ notation must work
	if serr := r.checkStatus(400,
		"In version 3.0 image should not be upscaled unless the ^ notation is used."); serr != nil {
		return serr
	}
	checkSize := func(want Size, sizeStr, message string) error {
		img, _, err := r.Client.GetImage(Params{Size: sizeStr})
		if err != nil {
			if serr := r.checkStatus(200, "Failed to retrieve upscaled image using ^ notation."); serr != nil {
				return serr
			}
			return r.fail("size", err.Error(), want, message)
		}
		if serr := r.check("size", imgSize(img), want, message); serr != nil {
			return serr
		}
		return checkGridSquares(r, img, want.W/10, want.W/10-13, 5, 3)
	}
	for _, c := range []struct {
		want    Size
		sizeStr string
		message string
	}{
		{Size{s, s}, fmt.Sprintf("^%d,%d", s, s), "Failed to get correct size for an image using the ^ notation"},
		{Size{s, s}, fmt.Sprintf("^,%d", s), "Failed to get correct size when asking for the height only using the ^ notation"},
		{Size{s, s}, fmt.Sprintf("^%d,", s), "Failed to get correct size when asking for the width only using the ^ notation"},
		{Size{1000, 1000}, "^max", "Failed to get max size while using the ^ notation"},
		{Size{2000, 2000}, "^pct:200", "Failed to get correct size when asking for the 200% size image and using the ^ notation"},
		{Size{500, 500}, "^!2000,500", "Failed to get correct size when trying to fit in a box !2000,500 using the ^ notation without upscaling"},
		{Size{2000, 2000}, "^!2000,3000", "Failed to get correct size when trying to fit in a box !2000,3000 using the ^ notation that requires upscaling."},
	} {
		if err := checkSize(c.want, c.sizeStr, c.message); err != nil {
			return err
		}
	}
	return nil
}
