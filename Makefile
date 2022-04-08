PROGRAM_NAME ?= device-flasher
PARALLEL_NAME ?= parallel-flasher
EXTENSIONS := linux exe darwin
NAMES := $(PROGRAM_NAME) $(PARALLEL_NAME)
PROGRAMS := $(foreach PROG,$(NAMES),$(foreach EXT,$(EXTENSIONS),$(PROG).$(EXT)))
VERSION := $(shell git describe --always --tags --dirty='-dirty')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -buildid=" -trimpath
AMD64_ARGS := GOARCH=amd64 CGO_ENABLED=0
ARM64_ARGS := GOARCH=arm64 CGO_ENABLED=0
# https://github.com/tpoechtrager/cctools-port
LIPO := /usr/bin/x86_64-apple-darwin-lipo

$(PROGRAM_NAME).%: TAGS := -tags release
$(PARALLEL_NAME).%: TAGS := -tags parallel

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

<<<<<<< HEAD
||||||| parent of baf5eeb (Build universal binaries for macOS)
# Debug variant of the default
$(PROGRAM_DEBUG_NAME).linux:
	$(COMMON_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).exe:
	$(COMMON_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).darwin:
	$(COMMON_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

=======
# Debug variant of the default
$(PROGRAM_DEBUG_NAME).linux:
	$(AMD64_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).exe:
	$(AMD64_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).darwin.amd64:
	$(AMD64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).darwin.arm64:
	$(ARM64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).darwin: $(PROGRAM_DEBUG_NAME).darwin.amd64 $(PROGRAM_DEBUG_NAME).darwin.arm64
	$(LIPO) -create $^ -o $@

>>>>>>> baf5eeb (Build universal binaries for macOS)
# With parallel (multi-device/model flashing enabled)
$(PARALLEL_NAME).linux:
	$(AMD64_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).exe:
	$(AMD64_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).darwin.amd64:
	$(AMD64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).darwin.arm64:
	$(ARM64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).darwin: $(PARALLEL_NAME).darwin.amd64 $(PARALLEL_NAME).darwin.arm64
	$(LIPO) -create $^ -o $@

<<<<<<< HEAD
||||||| parent of baf5eeb (Build universal binaries for macOS)
# Debug variant of parallel
$(PARALLEL_DEBUG_NAME).linux:
	$(COMMON_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).exe:
	$(COMMON_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).darwin:
	$(COMMON_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

=======
# Debug variant of parallel
$(PARALLEL_DEBUG_NAME).linux:
	$(AMD64_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).exe:
	$(AMD64_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).darwin.amd64:
	$(AMD64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).darwin.arm64:
	$(ARM64_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).darwin: $(PARALLEL_DEBUG_NAME).darwin.amd64 $(PARALLEL_DEBUG_NAME).darwin.arm64
	$(LIPO) -create $^ -o $@

>>>>>>> baf5eeb (Build universal binaries for macOS)
.PHONY: build
build: $(PROGRAMS)
	@echo Built $(VERSION)

clean:
	-rm $(PROGRAMS)
