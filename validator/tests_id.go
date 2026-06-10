package validator

import (
	"fmt"
	"math/rand"
	"strings"
)

func init() {
	register(&Test{
		Name: "id_basic", Label: "Image is returned", Level: 0, Category: 1,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{})
			data, err := r.Client.Fetch(url)
			if err != nil {
				return r.fail("status", err.Error(), 200, "Failed to retrieve url: "+url)
			}
			if err := r.checkStatus(200, ""); err != nil {
				return err
			}
			if _, _, err := DecodeImage(data); err != nil {
				return r.fail("status", err.Error(), 200, "Failed to retrieve url: "+url)
			}
			return nil
		},
	})

	register(&Test{
		Name: "id_squares", Label: "Correct image returned", Level: 0, Category: 1,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Format: "jpg"})
			data, err := r.Client.Fetch(url)
			if err != nil {
				return r.fail("status", err.Error(), 200, "Failed to retrieve url: "+url)
			}
			if err := r.checkStatus(200, ""); err != nil {
				return err
			}
			img, _, err := DecodeImage(data)
			if err != nil {
				return r.fail("status", r.Client.LastStatus, 200, "Failed to retrieve url: "+url)
			}
			// 100px reference squares; sample a 74px box inside each
			return checkGridSquares(r, img, 100, 74, 5, 4)
		},
	})

	register(&Test{
		Name: "id_escaped", Label: "Escaped characters processed", Level: 1, Category: 1,
		Versions: AllVersions,
		Run: func(r *Result) error {
			idf := strings.ReplaceAll(r.Client.Identifier, "-", "%2D")
			url := r.Client.MakeURL(Params{Identifier: idf})
			data, err := r.Client.Fetch(url)
			if err != nil {
				return r.fail("url-check", err.Error(), 404, "Failed to fetch url: "+url)
			}
			if err := r.checkStatus(200, ""); err != nil {
				return err
			}
			if _, _, err := DecodeImage(data); err != nil {
				return r.fail("url-check", err.Error(), 404, "Failed to fetch url: "+url)
			}
			return nil
		},
	})

	register(&Test{
		Name: "id_error_random", Label: "Random identifier gives 404", Level: 1, Category: 1,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Identifier: randomUUID()})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 404, "Failed to get random identifier from url: "+url+".")
			}
			return r.checkStatus(404, "")
		},
	})

	register(&Test{
		Name: "id_error_escapedslash", Label: "Forward slash gives 404", Level: 1, Category: 1,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Identifier: "a/b"})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 404, "Failed to get random identifier from url "+url+".")
			}
			return r.checkStatus(404, "")
		},
	})

	register(&Test{
		Name: "id_error_unescaped", Label: "Unescaped identifier gives 400", Level: 1, Category: 1,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Identifier: "[frob]"})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 400, "Failed to get random identifier from url: "+url+".")
			}
			return r.check("status", r.Client.LastStatus, []int{400, 404}, "")
		},
	})

	register(&Test{
		Name: "region_pixels", Label: "Region specified by pixels", Level: 1, Category: 2,
		Versions: AllVersions,
		Run: func(r *Result) error {
			match := 0
			for i := 0; i < 5; i++ {
				x, y := rand.Intn(10), rand.Intn(10)
				region := fmt.Sprintf("%d,%d,%d,%d", x*100+13, y*100+13, 74, 74)
				img, _, err := r.Client.GetImage(Params{Region: region})
				if err != nil {
					if serr := r.checkStatus(200, ""); serr != nil {
						return serr
					}
					return r.fail("General error", err.Error(), "No error", "Failed to check size due to: "+err.Error())
				}
				if r.TestSquare(img, x, y) {
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
		Name: "region_percent", Label: "Region specified by percent", Level: 2, Category: 2,
		Versions: AllVersions,
		Run: func(r *Result) error {
			match := 0
			for i := 0; i < 5; i++ {
				x, y := rand.Intn(10), rand.Intn(10)
				region := fmt.Sprintf("pct:%d,%d,9,9", x*10+1, y*10+1)
				img, _, err := r.Client.GetImage(Params{Region: region})
				if err != nil {
					if serr := r.checkStatus(200, ""); serr != nil {
						return serr
					}
					return r.fail("color-error", err.Error(), "No error", "Failed to check color due to "+err.Error())
				}
				if r.TestSquare(img, x, y) {
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
		Name: "region_square", Label: "Request a square region of the full image.",
		Level: 1, Levels: map[string]int{"3.0": 1, "2.1": 3, "2.1.1": 1}, Category: 3,
		Versions: []string{"3.0", "2.1", "2.1.1"},
		Run: func(r *Result) error {
			img, _, err := r.Client.GetImage(Params{Region: "square"})
			if serr := r.check("square-region", r.Client.LastStatus, 200,
				"A square region is mandatory for levels 1 and 2 in IIIF version 3.0."); serr != nil {
				return serr
			}
			if err != nil {
				return r.fail("square-region", err.Error(), "square image", "Failed to decode square region image.")
			}
			sz := imgSize(img)
			return r.check("square-region", sz.W, sz.H, "Square region returned a rectangle of unequal lengths.")
		},
	})

	register(&Test{
		Name: "region_error_random", Label: "Random region gives 400", Level: 1, Category: 2,
		Versions: AllVersions,
		Run: func(r *Result) error {
			url := r.Client.MakeURL(Params{Region: randomString(6)})
			if _, err := r.Client.Fetch(url); err != nil {
				return r.fail("url-check", err.Error(), 404, "Failed to get random region with url "+url+".")
			}
			return r.checkStatus(400, "")
		},
	})
}
