GO ?= go

check: lint test

lint:
	$(GO) vet ./...

test:
	$(GO) test -v ./...
