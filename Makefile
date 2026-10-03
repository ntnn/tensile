GO ?= go
TEST_ARGS ?= -race

TOOLS_DIR := $(CURDIR)/hack/tools

GOLANGCI_LINT_VER := 2.14.0
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint-$(GOLANGCI_LINT_VER)

check: lint test test-e2e

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) ./...
	cd ./test/e2e && $(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) ./...

.PHONY: lint-fix
lint-fix: override GOLANGCI_LINT_FLAGS := $(GOLANGCI_LINT_FLAGS) --fix
lint-fix: lint

.PHONY: test
test:
	$(GO) test $(TEST_ARGS) ./...

.PHONY: test-e2e
test-e2e:
	cd test/e2e && $(GO) test -count 1 $(TEST_ARGS) ./...

$(GOLANGCI_LINT):
	mkdir -p $(TOOLS_DIR)
	$(GO) tool github.com/ntnn/mindl download -tool golangci-lint -common -out $@ -version $(GOLANGCI_LINT_VER)
