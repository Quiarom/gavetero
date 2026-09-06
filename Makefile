GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST ?= bin
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
SIDECAR_DIR = cmd/gavetero/cmd/sidecars

.PHONY: build install-user uninstall-user fmt vet test check \
        frontend-install frontend-test frontend-build \
        dev dev-live clean

build:
	@mkdir -p $(DIST)
	@mkdir -p $(SIDECAR_DIR)
	@# 1. Build the sidecars and place them in the embed directory
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o $(SIDECAR_DIR)/router-core ./cmd/router-core
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o $(SIDECAR_DIR)/router-core-agent ./cmd/router-core-agent
	@# 2. Now build gavetero. go:embed picks up the binaries above
	#    and bakes them into the gavetero binary. The user only
	#    needs to install gavetero; the sidecars travel inside it.
	$(GO) build -ldflags="-X main.version=$(VERSION) -X github.com/Quiarom/router-core/cmd/gavetero/cmd.version=$(VERSION)" -o $(DIST)/gavetero ./cmd/gavetero
	@ln -sf gavetero $(DIST)/gvt
	@# 3. The transitional binaries are also written next to
	#    gavetero in $(DIST). The Tauri desktop build copies
	#    them into frontend/src-tauri/binaries/, and the
	#    engineering harness uses them directly.
	@cp $(SIDECAR_DIR)/router-core $(DIST)/router-core
	@cp $(SIDECAR_DIR)/router-core-agent $(DIST)/router-core-agent
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o $(DIST)/router-core-learn ./cmd/router-core-learn
	@chmod +x $(DIST)/router-core $(DIST)/router-core-agent
	@echo ""
	@echo "Built:"
	@echo "  $(DIST)/gavetero            -- user-facing, with embedded sidecars (~17MB)"
	@echo "  $(DIST)/gvt -> gavetero      -- shell alias"
	@echo "  $(DIST)/router-core         -- sidecar copy (Tauri / harness)"
	@echo "  $(DIST)/router-core-agent   -- sidecar copy (Tauri / harness)"
	@echo "  $(DIST)/router-core-learn   -- lab tool"

# Install the user-facing binary into ~/.local/bin.
# Because gavetero now embeds the sidecars, only one file is
# required for normal use. The sidecars are also written next
# to gavetero as a fallback for the live mode path.
install-user: build
	@mkdir -p $(BINDIR)
	@install -m 0755 $(DIST)/gavetero $(BINDIR)/gavetero
	@install -m 0755 $(DIST)/router-core $(BINDIR)/router-core
	@install -m 0755 $(DIST)/router-core-agent $(BINDIR)/router-core-agent
	@ln -sf gavetero $(BINDIR)/gvt
	@echo ""
	@echo "Installed in $(BINDIR):"
	@echo "  gvt          -> gavetero (user-facing, single binary with embedded sidecars)"
	@echo "  gavetero     (transitional: also used by gavetero sidecars)"
	@echo "  router-core  (fallback for gvt inspect --live when embed fails)"
	@echo "  router-core-agent (fallback for future gvt ask fallback path)"
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$(BINDIR)"; then \
		echo ""; \
		echo "Try in a new shell:"; \
		echo "  gvt version"; \
	else \
		echo ""; \
		echo "$(BINDIR) is not on PATH. Add this line to your shell rc:"; \
		echo "  export PATH=\"$(BINDIR):\$$PATH\""; \
		echo "Then open a new shell."; \
	fi

uninstall-user:
	@rm -f $(BINDIR)/gavetero $(BINDIR)/gvt $(BINDIR)/router-core $(BINDIR)/router-core-agent
	@echo "Removed gavetero, gvt, router-core, router-core-agent from $(BINDIR)"

fmt:
	gofmt -w .

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

check: fmt vet test frontend-test
	@echo "check OK"

frontend-install:
	cd frontend && npm ci

frontend-test:
	cd frontend && npm test

frontend-build:
	cd frontend && npm run build

dev:
	./scripts/dev.sh --mock

dev-live:
	./scripts/dev.sh --live

clean:
	rm -rf $(DIST) dist
