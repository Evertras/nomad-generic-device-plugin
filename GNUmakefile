PLUGIN_BINARY=generic-device
export GO111MODULE=on

default: build

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf generic-device launcher

.PHONY: build
build: device/*.go main.go go.mod go.sum
	go build -o ${PLUGIN_BINARY} .

.PHONY: eval
eval: deps build
	./launcher device ./${PLUGIN_BINARY} ./examples/config.hcl

.PHONY: fmt
fmt:
	@echo "==> Fixing source code with gofmt..."
	go fmt ./...

.PHONY: bootstrap
bootstrap: deps # install all dependencies

.PHONY: launcher
deps:  ## Install build and development dependencies
	@echo "==> Updating build dependencies..."
	go build github.com/hashicorp/nomad/plugins/shared/cmd/launcher

