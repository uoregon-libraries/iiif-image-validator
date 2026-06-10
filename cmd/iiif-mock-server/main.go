// Command iiif-mock-server serves the IIIF validator's reference image with
// version-appropriate Image API behavior. It exists for demos and for
// testing the validator itself (or other IIIF clients) without a real image
// server.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/uoregon-libraries/iiif-image-validator/validator/iiiftest"
)

func main() {
	addr := flag.String("addr", "localhost:8000", "listen address")
	version := flag.String("version", "2.0", "IIIF API version to emulate (1.0, 1.1, 2.0 or 3.0)")
	prefix := flag.String("prefix", "", "URL prefix to serve under (e.g. iiif)")
	flag.Parse()

	path := iiiftest.Identifier
	if *prefix != "" {
		path = strings.Trim(*prefix, "/") + "/" + path
	}
	fmt.Printf("Serving IIIF %s mock at http://%s/%s/info.json\n", *version, *addr, path)
	log.Fatal(http.ListenAndServe(*addr, iiiftest.NewServer(*version, strings.Trim(*prefix, "/"))))
}
