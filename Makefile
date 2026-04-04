GO ?= go

TOOLS_DIR := hack/tools

GOLANGCI_LINT_VER := 2.10.0
GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(TOOLS_DIR)/$(GOLANGCI_LINT_BIN)-$(GOLANGCI_LINT_VER)

check: lint test

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) ./...

.PHONY: lint-fix
lint-fix: override GOLANGCI_LINT_FLAGS := $(GOLANGCI_LINT_FLAGS) --fix
lint-fix: lint

.PHONY: test
test:
	$(GO) test -race -v ./...

$(GOLANGCI_LINT):
	mkdir -p $(TOOLS_DIR)
	$(GO) tool github.com/ntnn/mindl download \
		-url 'https://github.com/golangci/golangci-lint/releases/download/v{{.Version}}/golangci-lint-{{.Version}}-{{.OS}}-{{.Arch}}.{{.OSArchive}}' \
		-version $(GOLANGCI_LINT_VER) \
		-extract golangci-lint-{{.Version}}-{{.OS}}-{{.Arch}}/$(GOLANGCI_LINT_BIN){{.Exe}} \
		-out $@
