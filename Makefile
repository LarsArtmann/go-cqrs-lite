.PHONY: test build lint fmt imports check clean

test:
	GOWORK=off go test ./... -count=1 -v

test-race:
	GOWORK=off go test ./... -race -count=1

test-cover:
	GOWORK=off go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

build:
	GOWORK=off go build ./...

lint:
	GOWORK=off golangci-lint run --config .golangci.yml ./...

fmt:
	gofumpt -w .

imports:
	goimports -w .

check: fmt imports lint build test

clean:
	rm -f coverage.out
