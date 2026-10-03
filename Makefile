ifndef MAKE_VERSION
$(warning This Makefile requires GNU Make)
endif

MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

all: build samples

build:
	go build ./...

test:
	go test ./...

# Coverage of the library packages, excluding the examples.
cover:
	go test -coverprofile=cover.out $$(go list ./... | grep -v /examples/)
	go tool cover -func=cover.out

# Runs each fuzz target for FUZZTIME (e.g. `make fuzz FUZZTIME=10m`).
# `go test -fuzz` takes one target in one package at a time.
FUZZTIME ?= 60s
FUZZ_TARGETS := .:FuzzFiles ./lsp:FuzzByteSpanToRange ./lsp:FuzzRangeToByteSpan ./term:FuzzEmit

fuzz:
	@set -e; for t in $(FUZZ_TARGETS); do \
		echo "== $$t"; \
		go test $${t%%:*} -run '^$$' -fuzz "^$${t##*:}\$$" -fuzztime $(FUZZTIME); \
	done

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
