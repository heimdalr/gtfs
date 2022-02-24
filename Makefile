SHELL = /bin/bash

# just to be sure, add the path of the binary-based go installation.
PATH := /usr/local/go/bin:$(PATH)

# using the (above extended) path, query the GOPATH (i.e. the user's go path).
GOPATH := $(shell env PATH=$(PATH) go env GOPATH)

# add $GOPATH/bin to path
PATH := $(GOPATH)/bin:$(PATH)

# extract git version and hash using defaults
BUILD_GIT_HASH := $(shell git rev-parse HEAD 2>/dev/null || echo "0")
GIT_TAG := $(shell git describe --tags 2>/dev/null || echo "v0.0.0")
BUILD_VERSION := $(shell echo ${GIT_TAG} | grep -P -o '(?<=v)[0-9]+.[0-9]+.[0-9]')

# default golang flags (based on git infos)
LD_FLAGS := '-X main.buildVersion=$(BUILD_VERSION) -X main.buildGitHash=$(BUILD_GIT_HASH)'

# VBB data
VBB_ZIP_URL := https://www.vbb.de/fileadmin/user_upload/VBB/Dokumente/API-Datensaetze/gtfs-mastscharf/GTFS.zip
VBB_DATA_DIR := vbb
VBB_DB_FILE := vbb.db

GO_FILES := $(wildcard cmd/import/*.go cmd/trim/*.go *.go)
BIN_FILES := cmd/import/import cmd/trim/trim

all: $(BIN_FILES)

fmt:
	gofmt -s -w .

test:
	go test -p 4 -v ./...

lint:
	golangci-lint run

coverage:
	go test -coverprofile=c.out && go tool cover -html=c.out

# see: https://www.vbb.de/vbb-services/api-open-data/datensaetze/
# get or update VBB data
vbb:
	mkdir -p $(VBB_DATA_DIR)
	rm -f $(VBB_DATA_DIR)/*
	cd $(VBB_DATA_DIR) && wget $(VBB_ZIP_URL) && unzip GTFS.zip

# build the import tool
cmd/import/import: $(GO_FILES)
	go build \
	-ldflags $(LD_FLAGS) \
	-o $@ \
	heimdalr/gtfs/cmd/import

# import VBB data into SQLite DB
import: cmd/import/import $(VBB_DIR)
	$< $(VBB_DATA_DIR) $(VBB_DB_FILE)

# build the trim tool
cmd/trim/trim: $(GO_FILES)
	go build \
	-ldflags $(LD_FLAGS) \
	-o $@ \
	heimdalr/gtfs/cmd/trim

# trim DB to "S-Bahn Berlin GmbH"-agency
trim: cmd/trim/trim $(VBB_DB_FILE)
	$< $(VBB_DB_FILE) "S-Bahn"

clean:
	rm -f $(BIN_FILES)

.PHONY: all fmt test lint coverage vbb import trim clean