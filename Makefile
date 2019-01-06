NAME := go209
PKG := github.com/xntrik/go209

SHELL := /bin/bash
PREFIX?=$(shell pwd)
BUILDDIR := ${PREFIX}/cross
PLUGINDIR := ${PREFIX}/pkg/go209/modules
PLUGINS := $(wildcard $(PLUGINDIR)/*.go)
PLUGINSOUT := $(patsubst $(PLUGINDIR)/%.go, $(PREFIX)/%.so, $(PLUGINS))
.DEFAULT_GOAL := help

GO := go

all: clean fmt lint vet build ## Clean, fmt, lint, vet and build!

.PHONY: pluginsget
pluginsget: # run go get in the plugins folder before building
	@echo "Fetching plugin dependencies"
	cd $(PLUGINDIR); $(GO) get -d


.PHONY: buildplugins
buildplugins: pluginsget $(PLUGINSOUT) ## Build .so files from contents of pkg/go209/modules/*.go

$(PREFIX)/%.so: $(PLUGINDIR)/%.go
	@echo "+ $@"
ifndef STATIC_BUILD
	$(GO) build -buildmode=plugin -o $@ $<
else
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -buildmode=plugin -a -tags netgo -ldflags '-w' -o $@ $<
endif

.PHONY: build
build: buildplugins $(NAME) ## Builds the binary and plugins

static: # Build a static executable - don't forget to build the plugins statically as well
	@echo "+ $@"
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -a -tags netgo -ldflags '-w'

$(NAME): $(wildcard *.go) $(wildcard */*.go)
	@echo "+ $@"
	$(GO) build -o $(NAME) .

.PHONY: fmt
fmt: ## Verifies all files have been `gofmt`ed.
	@echo "+ $@"
	@if [[ ! -z "$(shell gofmt -s -l . | grep -v '.pb.go:' | grep -v '.twirp.go:' | grep -v vendor | tee /dev/stderr)" ]]; then \
		exit 1; \
	fi

.PHONY: lint
lint: ## Verifies `golint` passes.
	@echo "+ $@"
	@if [[ ! -z "$(shell golint ./... | grep -v '.pb.go:' | grep -v '.twirp.go:' | grep -v vendor | tee /dev/stderr)" ]]; then \
		exit 1; \
	fi

.PHONY: vet
vet: ## Verifies `go vet` passes.
	@echo "+ $@"
	@if [[ ! -z "$(shell $(GO) vet $(shell $(GO) list ./... | grep -v vendor | grep -v pkg/go209/modules) | tee /dev/stderr)" ]]; then \
		exit 1; \
	fi

.PHONY: image
image: clean ## Create docker image from the Dockerfile
	@docker build --rm --force-rm -t $(NAME) .

.PHONY: docker-compose-build
docker-compose-build: clean ## Build the docker compose
	@docker-compose build

.PHONY: docker-compose-up
docker-compose-up: ## Start the docker compose
	@docker-compose up

.PHONY: docker-compose-upd
docker-compose-upd: ## Start the docker compose in background mode
	@docker-compose up -d

.PHONY: clean
clean: ## Cleanup any build binaries or packages
	@echo "+ $@"
	$(RM) $(NAME)
	$(RM) -r $(BUILDDIR)
	$(RM) *.so

.PHONY: help
help:
	@echo -e "$$(grep -hE '^\S+:.*##' $(MAKEFILE_LIST) | sed -e 's/:.*##\s*/:/' -e 's/^\(.\+\):\(.*\)/\\x1b[36m\1\\x1b[m:\2/' | column -c2 -t -s :)"

check_defined = \
    $(strip $(foreach 1,$1, \
	$(call __check_defined,$1,$(strip $(value 2)))))

__check_defined = \
    $(if $(value $1),, \
    $(error Undefined $1$(if $2, ($2))$(if $(value @), \
    required by target `$@')))
