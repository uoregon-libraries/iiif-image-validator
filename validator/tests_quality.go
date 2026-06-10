package validator

import "image"

func init() {
	register(&Test{
		Name: "quality_color", Label: "Color quality", Level: 2, Category: 5,
		Versions: AllVersions,
		Run: func(r *Result) error {
			img, _, err := r.Client.GetImage(Params{Quality: "color"})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("quality", err.Error(), "color image", "Failed to retrieve color image.")
			}
			// PIL checked mode in (RGB, P); in Go a grayscale-encoded
			// response decodes to a Gray image
			switch img.(type) {
			case *image.Gray, *image.Gray16:
				return r.fail("quality", "grayscale", "color", "Color quality returned a grayscale-encoded image.")
			}
			r.Checks = append(r.Checks, "quality")
			return nil
		},
	})

	register(&Test{
		Name: "quality_grey", Label: "Gray/Grey quality", Level: 2, Category: 5,
		Versions: AllVersions,
		Run: func(r *Result) error {
			img, _, err := r.Client.GetImage(Params{Quality: "grey"})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("quality", err.Error(), "grey image", "Failed to retrieve grey image.")
			}
			switch img.(type) {
			case *image.Gray, *image.Gray16:
				return r.check("quality", 1, 1, "")
			}
			// check the vast majority of pixels have very similar r,g,b
			grey := countPixels(img, func(pr, pg, pb uint8) bool {
				return absDiff(pr, pg) < 5 && absDiff(pg, pb) < 5
			})
			if grey > 650000 {
				return r.check("quality", 1, 1, "")
			}
			return r.check("quality", 1, 0, "")
		},
	})

	register(&Test{
		Name: "quality_bitonal", Label: "Bitonal quality", Level: 2, Category: 5,
		Versions: AllVersions,
		Run: func(r *Result) error {
			img, _, err := r.Client.GetImage(Params{Quality: "bitonal"})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("quality", err.Error(), "bitonal image", "Failed to retrieve bitonal image.")
			}
			// check the vast majority of pixels are near-black or near-white
			ok := countPixels(img, func(pr, pg, pb uint8) bool {
				sum := int(pr) + int(pg) + int(pb)
				return sum < 15 || sum > 750
			})
			if ok > 650000 {
				return r.check("quality", 1, 1, "")
			}
			return r.check("quality", 1, 0, "")
		},
	})

	register(&Test{
		Name: "quality_error_random", Label: "Random quality gives 400", Level: 1, Category: 5,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Quality: randomString(6)})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 400, "Failed to get random quality from url: "+url+".")
			}
			return r.checkStatus(400, "")
		},
	})
}
