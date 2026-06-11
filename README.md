# IIIF Image API Validator (Go)

A Go port of the official [IIIF Image API validator][iiif-validator]. It runs the same suite of
tests as the Python validator — all 47 test modules, covering identifiers,
regions, sizes, rotation/mirroring, qualities, formats, info documents, and
HTTP behavior — against IIIF Image API versions 1.0, 1.1, 2.0 and 3.0.

Being a single static binary with no Python/Pillow/libmagic dependencies, it
is meant to be much easier to just grab and run quickly.

Your server must be serving the standard reference image as one of its
identifiers, exactly as with the Python validator.

[iiif-validator]: https://github.com/IIIF/image-validator

## Command line

```sh
make
./bin/iiif-validate -s images.example.org -p iiif/2 -i my-test-image -version 2.0 -level 2
```

Flags mirror the Python `iiif-validate.py`:

| Flag | Meaning |
|------|---------|
| `-s` / `-server` | server host, with port if not 80 (may include `http://` or `https://`) |
| `-p` / `-prefix` | URL prefix of the service, if any |
| `-i` / `-identifier` | identifier of the reference image on the server (required) |
| `-scheme` | `http` (default) or `https` |
| `-version` | IIIF Image API version: `1.0`, `1.1`, `2.0` (default) or `3.0` |
| `-level` | compliance level to test (default 1) |
| `-test` | run one named test instead of a level sweep (repeatable; see `-list`) |
| `-list` | list available tests for the chosen version |
| `-v` / `-verbose` | show URLs and per-check detail for passing tests |
| `-q` / `-quiet` | only print failures |
| `-json` | emit machine-readable JSON results on stdout |

The exit code is the number of failed tests, so it works directly as a CI
gate:

```sh
iiif-validate -s localhost:8182 -p iiif/2 -i validation.png -version 2.0 -level 2 || exit 1
```

## Setting up your server

Most of the validation tests check image *content* (colors of regions after
cropping, scaling, rotating, etc.), so they only pass when the server is
serving the standard 1000×1000 reference image. The bundled mock server
generates exactly that image (it's pixel-identical to the official CC0
reference image from the IIIF image-validator project), so you can pull a
lossless copy straight out of it with `curl`:

```sh
make
bin/iiif-mock-server -addr localhost:8000 &
curl -s -o validation.png http://localhost:8000/test-image/full/full/0/default.png
curl -s -o validation.jp2 http://localhost:8000/test-image/full/full/0/default.jp2
kill %1
```

(The PNG is encoded on the fly; the JP2 is a byte-for-byte copy of the official
file, since Go cannot encode JP2s without external dependencies. You can in
fact just copy the JP2 directly without the mock server; it lives in this repo
under `./validator/iiiftest/reference.jp2`)

Put it wherever your server reads images from (you can rename it; the
identifier just has to reach it), then run the validator against that
identifier.

For example, it's trivial to test the dockerized [RAIS image server][rais],
which serves images from `/var/local/images` under the `iiif` prefix on port
12415, using the filename as the identifier:

```sh
mkdir -p /tmp/rais-images
mv validation.jp2 /tmp/rais-images/
docker run --rm -p 12415:12415 -v /tmp/rais-images:/var/local/images uolibraries/rais:4-alpine
./bin/iiif-validate -s localhost:12415 -p iiif -i validation.jp2 -version 2.0 -level 2
```

[rais]: https://github.com/uoregon-libraries/rais-image-server

## Programmatic use from Go

```go
import "github.com/uoregon-libraries/iiif-image-validator/validator"

report := validator.Run(validator.Options{
        Server:     "images.example.org",
        Prefix:     "iiif/2",
        Identifier: "my-test-image",
        Version:    "2.0",
        Level:      2,
})
for _, r := range report.Results {
        if r.Err != nil {
                log.Printf("%s failed: %v", r.Name, r.Err)
        }
}
if report.Failures() > 0 {
        os.Exit(1)
}
```

Individual tests can be run with `validator.RunTest(name, client)`; the
available tests and their metadata (label, compliance level per version,
category) come from `validator.Tests(version)`.

## Mock server

The repository also ships a mock IIIF server which serves the reference image
with version-correct API behavior. It is used to test the validator itself,
and can be handy for demos or for testing other IIIF clients:

```sh
make
bin/iiif-mock-server -addr localhost:8000 -version 3.0 -prefix iiif
bin/iiif-validate -s localhost:8000 -p iiif -i test-image -version 3.0 -level 3
```

The same functionality is importable from
`github.com/uoregon-libraries/iiif-image-validator/validator/iiiftest`.

## Development

```sh
make test   # run the test suite
make vet    # static analysis
make bin    # build static binaries into bin/
make clean  # remove built binaries
```

The test suite includes an end-to-end run of every validator test against the
mock server for each supported API version.

## Differences from the Python validator

The goal is to exercise the same checks, not a line-for-line port:

- JP2 and PDF responses are verified by magic bytes instead of libmagic.
- Color-mode checks (e.g. `quality_color`) inspect the decoded image type
  rather than PIL mode strings.
- Error messages are similar but not byte-identical; warnings (e.g. 3.0
  `license`/`attribution`/`logo` checks) are reported as warnings, not
  failures, and don't affect the exit code.
- The bottle-based web service is not ported; this tool targets CLI and
  programmatic use.
