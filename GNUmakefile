PLUGIN_BINARY=generic-device
export GO111MODULE=on

default: $(PLUGIN_BINARY)

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf generic-device launcher
	rm -rf test/nomad

$(PLUGIN_BINARY): device/*.go main.go go.mod go.sum
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

deps:  ## Install build and development dependencies
	@echo "==> Updating build dependencies..."
	go build github.com/hashicorp/nomad/plugins/shared/cmd/launcher

.PHONY: nomad-test-server
nomad-test-server: ./bin/nomad ./test/server.hcl ./test/nomad/plugins/generic-device
	./bin/nomad agent -config ./test/server.hcl

NOMAD_VERSION := 1.2.6

# For now we only support Linux 64 bit and MacOS
ifeq ($(shell uname), Darwin)
OS_URL := darwin
else
OS_URL := linux
endif

./bin/nomad:
	@mkdir -p bin
	curl -o bin/nomad.zip \
		https://releases.hashicorp.com/nomad/$(NOMAD_VERSION)/nomad_$(NOMAD_VERSION)_$(OS_URL)_amd64.zip
	@cd bin && unzip nomad.zip
	@rm bin/nomad.zip

test/server.hcl: test/server.tpl.hcl
	@echo "Rendering server.tpl.hcl to server.hcl"
	sed "s%PWD%$${PWD}%g" test/server.tpl.hcl > test/server.hcl

test/nomad/plugins/generic-device: $(PLUGIN_BINARY)
	@mkdir -p test/nomad/plugins
	@cp $(PLUGIN_BINARY) test/nomad/plugins/

