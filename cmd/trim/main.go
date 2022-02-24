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
		fmt.Printf("usage: import [--version] [--help] <dbPath> <agency>\nflags:\n")
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
	dbPath := flag.Arg(0)
	agency := flag.Arg(1)

	// some argument validation
	if dbPath == "" {
		log.Fatal(errors.New("empty dbPath"))
	}
	if agency == "" {
		log.Fatal(errors.New("empty agency"))
	}

	// open gorm db
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
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

	// trim to agency
	trimResult, errTrim := gtfs.Trim(db, agency)
	if errTrim != nil {
		if errors.Is(errTrim, gorm.ErrRecordNotFound) {
			println(fmt.Sprintf("could not find an agency like '%s', not trimming", agency))
		} else {
			log.Fatal(fmt.Errorf("failed to trim DB: %w", errTrim))
		}
	}
	println(trimResult.String())

}
