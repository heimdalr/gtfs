# gtfs

<!--
[![run tests](https://github.com/heimdalr/dag/workflows/Run%20Tests/badge.svg?branch=master)](https://github.com/heimdalr/dag/actions?query=branch%3Amaster)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/heimdalr/dag)](https://pkg.go.dev/github.com/heimdalr/dag)
[![Go Report Card](https://goreportcard.com/badge/github.com/heimdalr/dag)](https://goreportcard.com/report/github.com/heimdalr/dag)
[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/heimdalr/dag)
-->

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
