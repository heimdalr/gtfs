# gtfs

[![Test](https://github.com/heimdalr/gtfs/actions/workflows/test.yml/badge.svg)](https://github.com/heimdalr/gtfs/actions/workflows/test.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/heimdalr/gtfs)](https://pkg.go.dev/github.com/heimdalr/gtfs)
[![Go Report Card](https://goreportcard.com/badge/github.com/heimdalr/gtfs)](https://goreportcard.com/report/github.com/heimdalr/gtfs)

Model and tooling for [General Transit Feed Specification (GTFS)]() data.

## Getting started

run (e.g.):

~~~~
make import
~~~~

to:

1. build the cli tool `./cmd/gtfs/gtfs`
2. download and extract [VBB GTFS data](https://www.vbb.de/vbb-services/api-open-data/datensaetze/) into `./vbb/`, and 
3. import the VBB GTFS data into the SQLite DB `./vbb.db`.

Then, mangle the DB using this package / model (`./gtfs.go`) with (e.g.) [GORM](https://gorm.io/).

## The CLI tool

to build (if not yet done) via: 

~~~~
make cmd/gtfs/gtfs
~~~~

and run (e.g.): 

~~~~
./cmd/gtfs/gtfs --help
~~~~
