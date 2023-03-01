GO ?= go

check: lint test examples

.PHONY: lint
lint:
	$(GO) vet ./...

.PHONY: test
test:
	$(GO) test -v ./...

EXAMPLES := $(wildcard examples/*)
examples: $(EXAMPLES)

.PHONY: $(EXAMPLES)
$(EXAMPLES):
	$(GO) run ./$@
