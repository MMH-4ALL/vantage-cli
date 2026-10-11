GO ?= go
VERSION ?= dev

.PHONY: build run test lint clean release
build:
	$(GO) build -trimpath -ldflags "-X main.version=$(VERSION)" -o bin/vantage ./cmd/vantage
run:
	$(GO) run ./cmd/vantage
test:
	$(GO) test ./...
lint:
	$(GO) vet ./...
	@test -z "$$($(GO) fmt ./...)" || (echo 'Formatting changed; review and commit it.'; exit 1)
clean:
	$(GO) clean
	rm -rf bin dist
release:
	$(GO) run scripts/release.go --version $(VERSION)
