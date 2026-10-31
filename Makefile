VERSION ?= 0.2.0
PROVIDER_HOSTNAME ?= igorovh
PROVIDER_NAMESPACE ?= anpi
NAME := terraform-provider-anpi
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)
EXE_EXT := $(if $(filter windows,$(OS)),.exe,)
BINARY := $(NAME)$(EXE_EXT)
PLUGIN_ROOT := $(if $(filter windows,$(OS)),$(HOME)/AppData/Roaming/terraform.d/plugins,$(HOME)/.terraform.d/plugins)
PLUGIN_DIR := $(PLUGIN_ROOT)/registry.terraform.io/$(PROVIDER_HOSTNAME)/$(PROVIDER_NAMESPACE)/$(VERSION)/$(OS)_$(ARCH)

.PHONY: build install-local test testacc fmt lint docs

build:
	mkdir -p bin
	go build -o bin/$(BINARY) .

install-local: build
	mkdir -p $(PLUGIN_DIR)
	cp bin/$(BINARY) $(PLUGIN_DIR)/$(NAME)_v$(VERSION)$(EXE_EXT)

test:
	go test ./...

# Needs a disposable anpi: ANPI_URL and ANPI_TOKEN (see README).
testacc:
	TF_ACC=1 go test ./internal/provider -run TestAcc -v -count=1

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './bin/*')

lint:
	go vet ./...

docs:
	@echo "Docs are maintained in docs/ and README.md"
