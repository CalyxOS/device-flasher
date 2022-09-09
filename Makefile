PROGRAM_NAME ?= device-flasher
PARALLEL_NAME ?= parallel-flasher
EXTENSIONS := linux exe darwin
NAMES := $(PROGRAM_NAME) $(PARALLEL_NAME)
PROGRAMS := $(foreach PROG,$(NAMES),$(foreach EXT,$(EXTENSIONS),$(PROG).$(EXT)))
VERSION := $(shell git describe --always --tags --dirty='-dirty')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -buildid=" -trimpath
COMMON_ARGS := GOARCH=amd64 CGO_ENABLED=0

$(PROGRAM_NAME).%: TAGS := -tags release
$(PARALLEL_NAME).%: TAGS := -tags parallel

all: clean build

# The default flasher, release build
$(PROGRAM_NAME).linux:
	$(COMMON_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).exe:
	$(COMMON_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PROGRAM_NAME).darwin:
	$(COMMON_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

# With parallel (multi-device/model flashing enabled)
$(PARALLEL_NAME).linux:
	$(COMMON_ARGS) GOOS=linux go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).exe:
	$(COMMON_ARGS) GOOS=windows go build $(TAGS) $(LDFLAGS) -o $@

$(PARALLEL_NAME).darwin:
	$(COMMON_ARGS) GOOS=darwin go build $(TAGS) $(LDFLAGS) -o $@

.PHONY: build
build: $(PROGRAMS)
	@echo Built $(VERSION)

clean:
	-rm $(PROGRAMS)
