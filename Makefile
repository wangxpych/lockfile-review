.PHONY: build coverage fmt fmt-check test verify vet

build:
	mkdir -p dist
	go build -trimpath -o dist/lockreview ./cmd/lockreview

coverage:
	go test -race -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

fmt:
	gofmt -w cmd internal

fmt-check:
	test -z "$$(gofmt -l cmd internal)"

test:
	go test -race ./...

vet:
	go vet ./...

verify: fmt-check vet test coverage build
