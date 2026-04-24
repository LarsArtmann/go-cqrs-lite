.PHONY: test test-core test-memory test-catalog test-middleware test-xtypes \
       build lint fmt imports check clean

MODULES = ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...

test:
	go test $(MODULES) -count=1 -v

test-race:
	go test $(MODULES) -race -count=1

test-cover:
	go test $(MODULES) -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

test-core:
	cd core && GOWORK=off go test ./... -count=1 -v

test-memory:
	cd memory && GOWORK=off go test ./... -count=1 -v

test-catalog:
	cd catalog && GOWORK=off go test ./... -count=1 -v

test-middleware:
	cd middleware && GOWORK=off go test ./... -count=1 -v

test-xtypes:
	cd xtypes && GOWORK=off go test ./... -count=1 -v

build:
	go build $(MODULES)

lint:
	go build $(MODULES)
	@for mod in core memory catalog middleware xtypes; do \
		echo "==> Linting $$mod"; \
		(cd $$mod && golangci-lint run --config ../.golangci.yml ./... 2>&1) || true; \
	done

fmt:
	gofumpt -w .

imports:
	goimports -w .

check: fmt imports lint build test

clean:
	rm -f coverage.out
