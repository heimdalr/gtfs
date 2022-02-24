package main

import (
	"errors"
	"flag"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"heimdalr/gtfs"
	"log"
	"os"
)

var (
	buildVersion = "to be set by linker"
	buildGitHash = "to be set by linker"
)

func main() {

	// init and parse flags
	help := flag.Bool("help", false, "help")
	version := flag.Bool("version", false, "version")
	flag.Usage = func() {
		fmt.Printf("usage: import [--version] [--help] <gtfsBasePath> <dbPath>\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// help
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// version info
	if *version {
		fmt.Printf("version: %s hash: %s", buildVersion, buildGitHash)
		os.Exit(0)
	}

	// get mandatory arguments
	if flag.NArg() != 2 {
		log.Fatal(errors.New("wrong number of arguments"))
	}
	gtfsBasePath := flag.Arg(0)
	dbPath := flag.Arg(1)

	// some argument validation
	if gtfsBasePath == "" {
		log.Fatal(errors.New("empty gtfsBasePath"))
	}
	if dbPath == "" {
		log.Fatal(errors.New("empty dbPath"))
	}

	// delete db-file, if it exists
	_, err := os.Stat(dbPath)
	if err == nil {
		if err = os.Remove(dbPath); err != nil {
			log.Fatal(fmt.Errorf("failed to remove old db file '%s'", dbPath))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

	// open gorm db
	var db *gorm.DB
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		log.Fatal(err)
	}

	// ensure tables matching our model
	err = gtfs.Migrate(db)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to migrate DB: %w", err))
	}

	// import CSV files
	importProgress := make(chan *gtfs.ImportItemsResult)
	go gtfs.Import(db, gtfsBasePath, importProgress)
	for importItemsResult := range importProgress {
		println(importItemsResult.String())
	}
}
