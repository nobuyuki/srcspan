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
samples: example term custom_files

example:
	go build -o ./bin/example ./examples/example
term:
	go build -o ./bin/term    ./examples/codespan/term
custom_files:
	go build -o ./bin/custom_files ./examples/codespan/custom_files

# (EOF)
