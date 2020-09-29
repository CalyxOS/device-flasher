PROGRAM_NAME ?= device-flasher
PROGRAM_DEBUG_NAME := $(PROGRAM_NAME)-debug
PARALLEL_NAME ?= parallel-flasher
PARALLEL_DEBUG_NAME := $(PARALLEL_NAME)-debug
EXTENSIONS := linux exe darwin
NAMES := $(PROGRAM_NAME) $(PROGRAM_DEBUG_NAME) $(PARALLEL_NAME) $(PARALLEL_DEBUG_NAME)
PROGRAMS := $(foreach PROG,$(NAMES),$(foreach EXT,$(EXTENSIONS),$(PROG).$(EXT)))
VERSION := $(shell git describe --always --tags --dirty='-dirty')
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

$(PROGRAM_NAME).%: TAGS := -tags release
$(PROGRAM_DEBUG_NAME).%: TAGS := -tags debug
$(PARALLEL_NAME).%: TAGS := -tags parallel
$(PARALLEL_DEBUG_NAME).%: TAGS := -tags parallel,debug

all: clean build

# The default flasher, release build
$(PROGRAM_NAME).linux:
	GOARCH=amd64 GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).exe:
	GOARCH=amd64 GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).darwin:
	GOARCH=amd64 GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

# Debug variant of the default
$(PROGRAM_DEBUG_NAME).linux:
	GOARCH=amd64 GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).exe:
	GOARCH=amd64 GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_DEBUG_NAME).darwin:
	GOARCH=amd64 GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

# With parallel (multi-device/model flashing enabled)
$(PARALLEL_NAME).linux:
	GOARCH=amd64 GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).exe:
	GOARCH=amd64 GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).darwin:
	GOARCH=amd64 GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

# Debug variant of parallel
$(PARALLEL_DEBUG_NAME).linux:
	GOARCH=amd64 GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).exe:
	GOARCH=amd64 GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_DEBUG_NAME).darwin:
	GOARCH=amd64 GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

.PHONY: build
build: $(PROGRAMS)
	@echo Built $(VERSION)

clean:
	-rm $(PROGRAMS)
