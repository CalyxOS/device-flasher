PROGRAM_NAME := device-flasher
OSES := linux windows darwin
PROGRAMS := $(foreach OS,$(OSES),$(PROGRAM_NAME).$(OS))

all: clean build

$(PROGRAM_NAME).linux:
	GOARCH=amd64 GOOS=linux go build -o $@

$(PROGRAM_NAME).windows:
	GOARCH=amd64 GOOS=windows go build -o $@

$(PROGRAM_NAME).darwin:
	GOARCH=amd64 GOOS=darwin go build -o $@

.PHONY: build
build: $(PROGRAMS)

clean:
	-rm $(PROGRAMS)
