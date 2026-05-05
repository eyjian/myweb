.PHONY: build build-ui dev test clean

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build: build-ui
	CGO_ENABLED=0 go build $(LDFLAGS) -o myweb ./cmd/myweb

build-ui:
	cd ui && npm run build

dev:
	go run ./cmd/myweb --dev --open=false

test:
	go test ./...

clean:
	rm -f myweb
	rm -rf ui/dist
