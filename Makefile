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

GO_FILES := $(wildcard cmd/gtfs/main.go cmd/gtfs/commands/*.go *.go)
BIN_FILES := cmd/gtfs/gtfs

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

# build the gtfs cmd
cmd/gtfs/gtfs: $(GO_FILES)
	go build \
	-ldflags $(LD_FLAGS) \
	-o $@ \
	heimdalr/gtfs/cmd/gtfs

# import VBB data into SQLite DB
import: cmd/gtfs/gtfs $(VBB_DIR)
	$< import $(VBB_DATA_DIR) $(VBB_DB_FILE)

# trim DB to "S-Bahn Berlin GmbH"-agency
trim: cmd/gtfs/gtfs $(VBB_DB_FILE)
	$< trim $(VBB_DB_FILE) "S-Bahn"

clean:
	rm -f $(BIN_FILES)

.PHONY: all fmt test lint coverage vbb import trim clean