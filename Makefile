ifndef MAKE_VERSION
$(warning This Makefile requires GNU Make)
endif

MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

all: build samples

build:
	go build ./...

test:
	go test ./...

clean:
	go clean -cache -modcache

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint --timeout 10m run  ./...


# Sample files.
#
# Builds every example under ./examples into ./bin, named after its
# directory. New examples are picked up without editing this file.
samples:
	go build -o ./bin/ ./examples/...

# (EOF)
