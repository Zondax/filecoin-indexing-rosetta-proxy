PACKAGE := github.com/zondax/filecoin-indexing-rosetta-proxy/tools
REVISION := $(shell git rev-parse --short HEAD)
ROSETTASDKVER := $(shell go list -m github.com/coinbase/rosetta-sdk-go | cut -d' ' -f2)
LOTUSVER := $(shell go list -m all | grep github.com/filecoin-project/lotus | awk '{print $$2}')
RETRYNUM := 10
ROSETTAPORT_CI := 8081
APPNAME := filecoin-indexing-rosetta-proxy

UNAME := $(shell uname)
ifeq ($(UNAME), Darwin)
export LIBRARY_PATH=$(shell brew --prefix hwloc)/lib
export LDFLAGS="-L$(LIBRARY_PATH)"
export LD_LIBRARY_PATH=$(LIBRARY_PATH)
export FFI_BUILD_FROM_SOURCE=0
endif

.PHONY: build
build: build_ffi
	go build -ldflags "-X $(PACKAGE).GitRevision=$(REVISION) -X $(PACKAGE).RosettaSDKVersion=$(ROSETTASDKVER) \
 	-X $(PACKAGE).LotusVersion=$(LOTUSVER)" -o $(APPNAME)

clean:
	go clean

build_ffi:
	make -C extern/filecoin-ffi

install_lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.1

check-modtidy:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

lint:
	golangci-lint --version
	golangci-lint run --timeout 5m
	golangci-lint fmt --diff

gitclean:
	git clean -xfd
	git submodule foreach --recursive git clean -xfd

.PHONY: mocks
mocks:
	@./tests/generate_mocks.sh