ifndef MAKE_VERSION
$(warning This Makefile requires GNU Make)
endif

MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

all: build samples

build:
	go build ./srcspan/...

test:
	go test ./srcspan/...

clean:
	go clean -cache -modcache

fmt:
	go fmt ./srcspan/...

vet:
	go vet ./srcspan/...

lint:
	golangci-lint --timeout 10m run  ./...


# Sample files.
#
# Builds every example under ./examples into ./bin, named after its
# directory. New examples are picked up without editing this file.
samples:
	go build -o ./bin/ ./examples/...

# (EOF)
