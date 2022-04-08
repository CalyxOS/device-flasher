PROGRAM_NAME ?= device-flasher
EXTENSIONS := linux exe darwin darwin.amd64 darwin.arm64
NAMES := $(PROGRAM_NAME)
PROGRAMS := $(foreach PROG,$(NAMES),$(foreach EXT,$(EXTENSIONS),$(PROG).$(EXT)))
VERSION := $(shell git describe --always --tags --dirty='-dirty')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -buildid=" -trimpath
AMD64_ARGS := GOARCH=amd64 CGO_ENABLED=0
ARM64_ARGS := GOARCH=arm64 CGO_ENABLED=0
# https://github.com/tpoechtrager/cctools-port
LIPO := /usr/bin/x86_64-apple-darwin-lip

$(PROGRAM_NAME).%: TAGS := -tags release

all: clean build

# The default flasher, release build
$(PROGRAM_NAME).linux:
	$(AMD64_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).exe:
	$(AMD64_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).darwin.amd64:
	$(AMD64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).darwin.arm64:
	$(ARM64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).darwin: $(PROGRAM_NAME).darwin.amd64 $(PROGRAM_NAME).darwin.arm64
	$(LIPO) -create $^ -o $@

.PHONY: build
build: $(PROGRAMS)
	@echo Built $(VERSION)

clean:
	-rm $(PROGRAMS)
