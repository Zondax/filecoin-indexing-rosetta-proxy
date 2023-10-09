PACKAGE := github.com/zondax/filecoin-indexing-rosetta-proxy/tools
REVISION := $(shell git rev-parse --short HEAD)
ROSETTASDKVER := $(shell go list -m all | grep github.com/coinbase/rosetta-sdk-go | awk '{print $$2}')
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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2

check-modtidy:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

lint:
	golangci-lint --version
	golangci-lint run -E gofmt -E gosec -E goconst -E gocritic --timeout 5m

gitclean:
	git clean -xfd
	git submodule foreach --recursive git clean -xfd
