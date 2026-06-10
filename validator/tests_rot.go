package validator

import (
	"fmt"
	"image"
	"math/rand"
)

// rotCorners checks the two corner squares of a rotated/mirrored full image:
// the top-left 64px box must match reference square (tlx,tly) and the
// bottom-right box (brx,bry).
func rotCorners(r *Result, img image.Image, errType string, tlx, tly, brx, bry int) error {
	w := imgSize(img).W
	if w < 999 || w > 1001 {
		return r.fail("size", imgSize(img), Size{1000, 1000}, "")
	}
	sqr := crop(img, 12, 12, 76, 76)
	if !r.TestSquare(sqr, tlx, tly) {
		return r.fail(errType, 1, ColorInfo[9][9], "")
	}
	sqr = crop(img, 912, 912, 976, 976)
	if !r.TestSquare(sqr, brx, bry) {
		return r.fail(errType, 1, ColorInfo[0][0], "")
	}
	return nil
}

func init() {
	register(&Test{
		Name: "rot_full_basic", Label: "Rotation by 90 degree values",
		Level: 2, Levels: map[string]int{"3.0": 2, "2.0": 2, "1.0": 1, "1.1": 1}, Category: 4,
		Versions: AllVersions,
		Run: func(r *Result) error {
			for _, c := range []struct {
				rotation           string
				tlx, tly, brx, bry int
			}{
				{"180", 9, 9, 0, 0},
				{"90", 0, 9, 9, 0},
				{"270", 9, 0, 0, 9},
			} {
				img, _, err := r.Client.GetImage(Params{Rotation: c.rotation})
				if err != nil {
					if serr := r.checkStatus(200, ""); serr != nil {
						return serr
					}
					return r.fail("rotation", err.Error(), "rotated image", "Failed to retrieve rotated image.")
				}
				if err := rotCorners(r, img, "color", c.tlx, c.tly, c.brx, c.bry); err != nil {
					return err
				}
			}
			return nil
		},
	})

	register(&Test{
		Name: "rot_region_basic", Label: "Rotation of region by 90 degree values",
		Level: 2, Levels: map[string]int{"3.0": 2, "2.0": 2, "1.0": 1, "1.1": 1}, Category: 4,
		Versions: AllVersions,
		Run: func(r *Result) error {
			const s = 76
			for i := 0; i < 4; i++ {
				x, y := rand.Intn(10), rand.Intn(10)
				img, _, err := r.Client.GetImage(Params{
					Rotation: "180",
					Region:   fmt.Sprintf("%d,%d,%d,%d", x*100+13, y*100+13, s, s),
				})
				if err != nil {
					if serr := r.checkStatus(200, ""); serr != nil {
						return serr
					}
					return r.fail("rotation", err.Error(), "rotated image", "Failed to retrieve rotated region.")
				}
				w := imgSize(img).W
				if w < s-1 || w > s+1 { // leeway for rotation
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
		Name: "rot_full_non90", Label: "Rotation by non 90 degree values", Level: 3, Category: 4,
		Versions: AllVersions,
		Run: func(r *Result) error {
			rot := rand.Intn(359) + 1
			_, _, err := r.Client.GetImage(Params{Rotation: fmt.Sprint(rot)})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("rotation", err.Error(), "rotated image", "Failed to retrieve rotated image.")
			}
			// not sure how to test, other than we got an image
			return nil
		},
	})

	register(&Test{
		Name: "rot_region_non90", Label: "Rotation by non 90 degree values", Level: 3, Category: 4,
		Versions: AllVersions,
		Run: func(r *Result) error {
			for i := 0; i < 4; i++ {
				rot := rand.Intn(359) + 1
				x, y := rand.Intn(10), rand.Intn(10)
				_, _, err := r.Client.GetImage(Params{
					Rotation: fmt.Sprint(rot),
					Region:   fmt.Sprintf("%d,%d,100,100", x*100, y*100),
				})
				if err != nil {
					if serr := r.checkStatus(200, ""); serr != nil {
						return serr
					}
					return r.fail("rotation", err.Error(), "rotated image", "Failed to retrieve rotated region.")
				}
			}
			return nil
		},
	})

	register(&Test{
		Name: "rot_mirror", Label: "Mirroring", Level: 3, Category: 4,
		Versions: []string{"2.0", "3.0"},
		Run: func(r *Result) error {
			img, _, err := r.Client.GetImage(Params{Rotation: "!0"})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("mirror", err.Error(), "mirrored image", "Failed to retrieve mirrored image.")
			}
			return rotCorners(r, img, "mirror", 9, 0, 0, 9)
		},
	})

	register(&Test{
		Name: "rot_mirror_180", Label: "Mirroring plus 180 rotation", Level: 3, Category: 4,
		Versions: []string{"2.0", "3.0"},
		Run: func(r *Result) error {
			img, _, err := r.Client.GetImage(Params{Rotation: "!180"})
			if err != nil {
				if serr := r.checkStatus(200, ""); serr != nil {
					return serr
				}
				return r.fail("mirror", err.Error(), "mirrored image", "Failed to retrieve mirrored image.")
			}
			return rotCorners(r, img, "mirror", 0, 9, 9, 0)
		},
	})

	register(&Test{
		Name: "rot_error_random", Label: "Random rotation gives 400", Level: 1, Category: 4,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Rotation: randomString(4)})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 404, "Failed to get random rotation from url: "+url+".")
			}
			return r.checkStatus(400, "")
		},
	})
}
