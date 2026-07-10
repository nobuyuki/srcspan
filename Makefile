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
samples: example term

example:
	go build -o ./bin/example ./examples/example
term:
	go build -o ./bin/term    ./examples/codespan/term

# (EOF)
