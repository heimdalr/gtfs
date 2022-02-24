package gtfs_test

import (
	"fmt"
	"github.com/heimdalr/gtfs"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Example() {

	const dbPath = "_fixture/test.db"

	// open the DB
	db, _ := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})

	// SELECT * FROM agencies WHERE id = 1;
	agency := gtfs.Agency{}
	db.First(&agency, "id = ?", 1)
	fmt.Println(agency)

	// Output:
	// {1 S-Bahn Berlin GmbH https://sbahn.berlin/}
}
