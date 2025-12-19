.PHONY: build install clean test help

# Variables
BINARY_NAME=staticup
GO=go
INSTALL_PATH=/usr/local/bin

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	$(GO) build -o $(BINARY_NAME) .

install: build ## Build and install the binary to /usr/local/bin (requires sudo)
	sudo mv $(BINARY_NAME) $(INSTALL_PATH)/

clean: ## Remove built binary
	rm -f $(BINARY_NAME)

test: ## Run tests
	$(GO) test -v ./...

fmt: ## Format the code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

run: build ## Build and run the tool (use ARGS variable for arguments)
	./$(BINARY_NAME) $(ARGS)
