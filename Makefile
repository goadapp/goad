SHELL := /bin/bash

# name of the binary created
TARGET := goad

# Prepend our vendor directory to the system GOPATH
# so that import path resolution will prioritize
# our third party snapshots.
GOPATH := ${PWD}/vendor:${GOPATH}
export GOPATH

# These will be provided to the target
VERSION := 1.4.1
BUILD := `git rev-parse HEAD`

# Timestamp of last commit to allow for reproducable builds
TIMESTAMP := `git log -1 --date=format:%Y%m%d%H%M --pretty=format:%cd`

# Use linker flags to provide version/build settings to the target
LDFLAGS = -ldflags "-X=github.com/goadapp/goad/version.version=$(VERSION) -X=github.com/goadapp/goad/version.build=$(BUILD)"

# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

# go source folders to test
TEST = $(shell go list ./... | grep -v /vendor/)

# $(GO-BUILD) command
GO-BUILD = go build $(LDFLAGS)

# $(ZIP) command ignoring timestamps and using UTC timezone
ZIP = TZ=UTC zip -jrX

.PHONY: lambda bindata clean all-zip all linux osx windows check fmt test install uninstall

all: osx linux windows

test: bindata
	@go test $(TEST)

lambda:
	@GOOS=linux GOARCH=amd64 $(GO-BUILD) -o data/lambda/goad-lambda ./lambda
	@find data/lambda -exec touch -t $(TIMESTAMP) {} \; # strip timestamp
	@$(ZIP) data/lambda data/lambda

bindata: lambda
	@go get github.com/jteeuwen/go-bindata/...
	@go-bindata -modtime $(TIMESTAMP) -nocompress -pkg infrastructure -o infrastructure/bindata.go data/lambda.zip

linux: bindata
	@GOOS=linux GOARCH=amd64 $(GO-BUILD) -o build/linux/x86-64/$(TARGET)
	@GOOS=linux GOARCH=386 $(GO-BUILD) -o build/linux/x86/$(TARGET)

osx: bindata
	@GOOS=darwin GOARCH=amd64 $(GO-BUILD) -o build/osx/x86-64/$(TARGET)

windows: bindata
	@GOOS=windows GOARCH=amd64 $(GO-BUILD) -o build/windows/x86-64/$(TARGET)
	@GOOS=windows GOARCH=386 $(GO-BUILD) -o build/windows/x86/$(TARGET)

clean:
	@rm -rf data/lambda/goad-lambda
	@rm -rf data/lambda.zip
	@rm -rf build
	@rm -rf infrastructure/bindata.go

build: bindata
	@$(GO-BUILD) $(LDFLAGS) -o build/$(TARGET)

install: bindata
	@go install $(LDFLAGS)

uninstall: clean
	@go clean -i

fmt:
	@gofmt -l -w $(SRC)

simplify:
	@gofmt -s -l -w $(SRC)

check:
	@test -z $(shell gofmt -l main.go | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"
	@for d in $$(go list ./... | grep -v /vendor/); do golint $${d}; done
	@go tool vet ${SRC}

all-zip: all
	@mkdir -p ./build/zip
	@find build -exec touch -t $(TIMESTAMP) {} \; # strip timestamp
	@$(ZIP) ./build/zip/goad-osx-x86-64 ./build/osx/x86-64/goad
	@$(ZIP) ./build/zip/goad-linux-x86-64 ./build/linux/x86-64/goad
	@$(ZIP) ./build/zip/goad-linux-x86 ./build/linux/x86/goad
	@$(ZIP) ./build/zip/goad-windows-x86-64 ./build/windows/x86-64/goad
	@$(ZIP) ./build/zip/goad-windows-x86 ./build/windows/x86/goad
