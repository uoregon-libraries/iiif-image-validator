.PHONY: all bin test vet fmt clean

BUILD := CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"

all: bin

bin:
	$(BUILD) -o bin/iiif-validate ./cmd/iiif-validate
	$(BUILD) -o bin/iiif-mock-server ./cmd/iiif-mock-server

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin
