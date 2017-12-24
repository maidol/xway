.PHONY: all

PKGS := $(shell go list ./... | grep -v '/vendor/')
TXT_FILES := $(shell find * -type f -not -path 'vendor/**')

default: clean misspell vet test

test: clean
	go test -race -cover $(PKGS)

test-verbose: clean
	go test -v -race -cover $(PKGS)

dependencies:
	dep ensure -v

clean:
	find . -name flymake_* -delete
	rm -f cover.out

lint:
	echo "golint:"
	golint -set_exit_status $(PKGS)

vet:
	go vet $(PKGS)

checks: vet lint check-fmt
	echo "staticcheck:"
	staticcheck $(PKGS)
	echo "gosimple:"
	gosimple $(PKGS)

check-fmt: SHELL := /bin/bash
check-fmt:
	diff -u <(echo -n) <(gofmt -d $(GOFILES))

misspell:
	misspell -source=text -error $(TXT_FILES)

test-package: clean
	go test -v ./$(p)

test-grep-package: clean
	go test -v ./$(p) -check.f=$(e)

cover-package: clean
	go test -v ./$(p)  -coverprofile=/tmp/coverage.out
	go tool cover -html=/tmp/coverage.out

sloccount:
	 find . -path ./vendor -prune -o -name "*.go" -print0 | xargs -0 wc -l
