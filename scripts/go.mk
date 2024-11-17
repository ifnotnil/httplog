GO_EXEC ?= go
export GO_EXEC

MODULE := $(shell cat go.mod | grep -e "^module" | sed "s/^module //")

GO_PACKAGES = $(GO_EXEC) list -tags='$(TAGS)' -mod=readonly ./...
GO_FOLDERS = $(GO_EXEC) list -tags='$(TAGS)' -mod=readonly -f '{{ .Dir }}' ./...
GO_FILES = find . -type f -name '*.go'

export GO111MODULE := on
#export GOFLAGS := -mod=readonly
GOPATH := $(shell go env GOPATH)
GO_VER := $(shell go env GOVERSION)
BUILD_OUTPUT ?= $(CURDIR)/output


.PHONY: mod
mod:
	$(GO_EXEC) mod tidy -go=1.22
	$(GO_EXEC) mod verify

# https://go.dev/ref/mod#go-get
# -u flag tells go get to upgrade modules
# -t flag tells go get to consider modules needed to build tests of packages named on the command line.
# When -t and -u are used together, go get will update test dependencies as well.
.PHONY: go-deps-upgrade
go-deps-upgrade:
	$(GO_EXEC) get -u -t ./...
	$(GO_EXEC) mod tidy -go=1.23
	$(GO_EXEC) mod vendor

.PHONY: test
test:
	CGO_ENABLED=1 $(GO_EXEC) test -timeout 60s -race -tags='$(TAGS)' -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: test-n-read
test-n-read: test
	@$(GO_EXEC) tool cover -func coverage.txt

.PHONY: bench
bench:
	CGO_ENABLED=1 $(GO_EXEC) test -benchmem -run=^$$ -mod=readonly -count=1 -v -race -bench=. ./...

