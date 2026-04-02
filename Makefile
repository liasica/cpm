APP_NAME := cpm
VERSION  := $(shell git log -1 --format='%cd-%h' --date=format:'%Y%m%d' 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X main.version=$(VERSION)

.PHONY: build install clean test lint

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/cpm

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/cpm

clean:
	rm -rf bin/ dist/

test:
	go test ./...

lint:
	golangci-lint run ./...
