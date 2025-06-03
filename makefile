# For local use in this makefile. This does not export to sub-processes.
-include .env.default.properties
-include $(or $(CONF),.)/.env.properties

MAKEFLAGS        := --silent --always-make
MAKE_CONC        := $(MAKE) -j 128 CONC=true clear=$(or $(clear),false)

VERB_SHORT       ?= $(if $(filter false,$(verb)),,-v)
CLEAR_SHORT      ?= $(if $(filter false,$(clear)),,-c)

APP              ?= got
TAR              ?= .tar
GO_CMD           ?= $(TAR)/$(APP)
GO_CMD_LINUX     ?= $(GO_CMD)_linux
GO_CMD_SRC       ?= .

GO_PKG           ?= $(or $(pkg),./...)
GO_FLAGS         ?= -tags=$(tags) -mod=mod -buildvcs=false
GO_RUN_ARGS      ?= $(GO_FLAGS) $(GO_CMD_SRC) $(args)
GO_VET_FLAGS     ?= -composites=false
GO_WATCH_FLAGS   ?= $(and $(pkg),-w=$(pkg))

GO_TEST_FAIL     ?= $(if $(filter false,$(fail)),,-failfast)
GO_TEST_SHORT    ?= $(if $(filter true,$(short)), -short,)
GO_TEST_FLAGS    ?= -count=1 $(GO_FLAGS) $(VERB_SHORT) $(GO_TEST_FAIL) $(GO_TEST_SHORT)
GO_TEST_PATTERNS ?= -run="$(run)"
GO_TEST_ARGS     ?= $(GO_PKG) $(GO_TEST_FLAGS) $(GO_TEST_PATTERNS)

# Disable raw mode because it interferes with our TTY detection.
GOW_HOTKEYS := -r=false

# Repo: https://github.com/mitranim/gow.
# Install: `go install github.com/mitranim/gow@latest`.
GOW ?= gow $(CLEAR_SHORT) $(VERB_SHORT) $(GOW_HOTKEYS)

# Repo: https://github.com/mattgreen/watchexec.
# Install: `brew install watchexec`.
WATCH ?= watchexec $(CLEAR_SHORT) -d=0 -r -n --stop-timeout=1

OK = echo [$@] ok

# TODO: if appropriate executable does not exist, print install instructions.
ifeq ($(OS),Windows_NT)
	GO_WATCH ?= $(WATCH) $(GO_WATCH_FLAGS) -- go
else
	GO_WATCH ?= $(GOW) $(GO_WATCH_FLAGS)
endif

ifeq ($(OS),Windows_NT)
	RM_DIR = if exist "$(1)" rmdir /s /q "$(1)"
else
	RM_DIR = rm -rf "$(1)"
endif

ifeq ($(OS),Windows_NT)
	CP_INNER = if exist "$(1)" copy "$(1)"\* "$(2)" >nul
else
	CP_INNER = if [ -d "$(1)" ]; then cp -r "$(1)"/* "$(2)" ; fi
endif

ifeq ($(OS),Windows_NT)
	CP_DIR = if exist "$(1)" copy "$(1)" "$(2)" >nul
else
	CP_DIR = if [ -d "$(1)" ]; then cp -r "$(1)" "$(2)" ; fi
endif

default: go.run.w

go.run.w: # Run in watch mode
	$(GO_WATCH) run $(GO_RUN_ARGS)

go.run: # Run once
	go run $(GO_RUN_ARGS)

go.test.w: # Run tests in watch mode
	$(eval export)
	$(GO_WATCH) test $(GO_TEST_ARGS)

go.test: # Run tests once
	$(eval export)
	go test $(GO_TEST_ARGS)

go.vet.w: # Run `go vet` in watch mode
	$(GO_WATCH) vet $(GO_FLAGS) $(GO_VET_FLAGS) $(GO_PKG)

go.vet: # Run `go vet` once
	go vet $(GO_FLAGS) $(GO_VET_FLAGS) $(GO_PKG)
	$(OK)

go.build: # Build executable for current platform
	go build $(GO_FLAGS) -o=$(GO_CMD) $(GO_CMD_SRC)

go.build.linux: # Build executable for Linux
	GOOS=linux GOARCH=amd64 go build -o $(GO_CMD_LINUX) $(GO_CMD_SRC)

go.install:
	$(MAKE) run cmd="go install github.com/yury-egorenkov/got@latest"

run:
	echo $(cmd) && $(cmd) && echo "ok"

# TODO: keep command comments, and align them vertically.
#
# Note that if we do that, `uniq` will no longer dedup lines for commands whose
# names are repeated, usually with `<cmd_name>: export <VAR> ...`. We'd have to
# skip/ignore those lines.
help:  # Print help
	echo "Available commands are listed below"
	echo "Show this help: make help"
	echo "Show command definition: make -n <command>"
	echo
	for val in $(MAKEFILE_LIST); do grep -E '^\S+:' $$val; done | sed 's/:.*//' | uniq
