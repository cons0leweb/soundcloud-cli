.PHONY: build test vet check clean

BINARY := soundcloud
VERSION ?= dev
GOFLAGS := -buildvcs=false
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/soundcloud

test:
	go test $(GOFLAGS) -race ./...

vet:
	go vet $(GOFLAGS) ./...

check: vet test

clean:
	go clean
	rm -f $(BINARY) coverage.out
