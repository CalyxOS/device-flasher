PROGRAM_NAME ?= device-flasher
EXTENSIONS := linux exe darwin
PROGRAMS := $(foreach EXT,$(EXTENSIONS),$(PROGRAM_NAME).$(EXT))
VERSION := $(shell git describe --always --tags --dirty='-dirty')
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

all: clean build

$(PROGRAM_NAME).linux:
	@echo $(VERSION)
	GOARCH=amd64 GOOS=linux go build $(LDFLAGS) -o $@

$(PROGRAM_NAME).exe:
	GOARCH=amd64 GOOS=windows go build $(LDFLAGS) -o $@

$(PROGRAM_NAME).darwin:
	GOARCH=amd64 GOOS=darwin go build $(LDFLAGS) -o $@

.PHONY: build
build: $(PROGRAMS)

clean:
	-rm $(PROGRAMS)
