PROGRAM_NAME ?= device-flasher
OSES := linux windows darwin
PROGRAMS := $(foreach OS,$(OSES),$(PROGRAM_NAME).$(OS))
VERSION := $(shell git describe --always --tags --dirty='-dirty')
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

all: clean build

$(PROGRAM_NAME).linux:
	@echo $(VERSION)
	GOARCH=amd64 GOOS=linux go build $(LDFLAGS) -o $@

$(PROGRAM_NAME).windows:
	GOARCH=amd64 GOOS=windows go build $(LDFLAGS) -o $@

$(PROGRAM_NAME).darwin:
	GOARCH=amd64 GOOS=darwin go build $(LDFLAGS) -o $@

.PHONY: build
build: $(PROGRAMS)

clean:
	-rm $(PROGRAMS)
