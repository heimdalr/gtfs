# gtfs

[![Test](https://github.com/heimdalr/gtfs/actions/workflows/test.yml/badge.svg)](https://github.com/heimdalr/gtfs/actions/workflows/test.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/heimdalr/gtfs)](https://pkg.go.dev/github.com/heimdalr/gtfs)
[![Go Report Card](https://goreportcard.com/badge/github.com/heimdalr/gtfs)](https://goreportcard.com/report/github.com/heimdalr/gtfs)

Model and tooling for [General Transit Feed Specification (GTFS)]() data.

## Getting started

### Using the CLI tool

Get, build and install the `gtfs` binary by running:

~~~~
go install github.com/heimdalr/gtfs/cmd/gtfs@latest
~~~~

If all goes well, the above installed `~/go/bin/gtfs`. 

Now download and extract (e.g.) the [VBB GTFS data](https://www.vbb.de/vbb-services/api-open-data/datensaetze/) to `./vbb/`
by running:

~~~~
mkdir ./vbb
cd ./vbb 
wget https://www.vbb.de/fileadmin/user_upload/VBB/Dokumente/API-Datensaetze/gtfs-mastscharf/GTFS.zip
unzip GTFS.zip
~~~~

Finally, run: 

~~~~
gtfs import ./vbb ./vbb.db
~~~~

to import the VBB GTFS CSV files within `./vbb/` into the SQLite DB file `./vbb.db`.

### Using the Model

   
Now, using this package on the DB we populated in the previous step, we may build and run the following: 

~~~~
package main

import (
	"fmt"
	"github.com/heimdalr/gtfs"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {

	const dbPath = "vbb.db"

	// open the DB
	db, _ := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})

	// SELECT * FROM agencies WHERE id = 1;
	agency := gtfs.Agency{}
	db.First(&agency, "id = ?", 1)
	fmt.Println(agency)
}

~~~~

to query for the agency with the ID "1":

~~~~
{1 S-Bahn Berlin GmbH https://sbahn.berlin/}
~~~~

