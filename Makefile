GO ?= go

check: lint test

.PHONY: lint
lint:
	$(GO) vet ./...

.PHONY: test
test:
	$(GO) test -v ./...
