
# enable modules
GO111MODULE=on
# disable CGO by default for static binaries
CGO_ENABLED=0
# Use current hash to tag the container images
COMMIT:=$(shell git rev-parse --short HEAD 2>/dev/null)
TAG_GENERATE_IMAGE?=$(COMMIT)
export GO111MODULE CGO_ENABLED TAG_GENERATE_IMAGE

build:
	mkdir -p bin
	go build -o bin

image: build
	docker build --no-cache -t aojea/funnyips:test -f Dockerfile .

test:
	go test ./...

clean:
	rm -rf bin

all: build