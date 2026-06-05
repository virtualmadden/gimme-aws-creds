BINARY := gimme-aws-creds
VERSION ?= dev
LDFLAGS := -ldflags "-s -w -X github.com/virtualmadden/gimme-aws-creds/internal/cli.version=$(VERSION)"
GOBIN ?= $(shell go env GOBIN)
GOPATH_BIN ?= $(shell go env GOPATH)/bin
INSTALL_DIR ?= $(if $(GOBIN),$(GOBIN),$(GOPATH_BIN))

.PHONY: build test lint clean install install-path

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/gimme-aws-creds

install:
	go install $(LDFLAGS) ./cmd/gimme-aws-creds
	@echo ""
	@echo "Installed $(BINARY) to $(INSTALL_DIR)/$(BINARY)"
	@echo ""
	@case ":$$PATH:" in \
		*":$(INSTALL_DIR):"*) \
			echo "$(INSTALL_DIR) is already on your PATH." ;; \
		*) \
			echo "Add this to your shell profile (~/.zshrc or ~/.bashrc):"; \
			echo '  export PATH="$$PATH:$(INSTALL_DIR)"'; \
			echo ""; \
			echo "Or run once in this shell:"; \
			echo '  export PATH="$$PATH:$(INSTALL_DIR)"' ;; \
	esac

# Copy the built binary to a directory already on PATH (default: /usr/local/bin).
install-path: build
	install -d "$(DESTDIR)/usr/local/bin"
	install -m 755 bin/$(BINARY) "$(DESTDIR)/usr/local/bin/$(BINARY)"
	@echo "Installed $(BINARY) to $(DESTDIR)/usr/local/bin/$(BINARY)"

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/ dist/
