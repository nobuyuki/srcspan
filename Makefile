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


# Sample files.
#
samples: simple example

simple:
	go build -o ./bin/simple  ./examples/simple
example:
	go build -o ./bin/example ./examples/example

# (EOF)
