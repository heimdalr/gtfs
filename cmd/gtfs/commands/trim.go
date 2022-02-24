package commands

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"heimdalr/gtfs"
	"log"
	"strings"
	"time"
)

const (

	// statement to remove all agencies not like a given name
	delAgencyStmt = `
DELETE
FROM
	agencies
WHERE
	id <> ?;
`

	// statement to remove all routes not belonging to any of the known agencies
	delRoutesStmt = `
DELETE
FROM
	routes
WHERE agency_id NOT IN (
	SELECT DISTINCT id
	FROM
		agencies);
`

	// statement to remove all trips not belonging to any of the known routes
	delTripsStmt = `
DELETE
FROM
	trips
WHERE route_id NOT IN (
	SELECT DISTINCT id
	FROM
		routes);
`

	// statement to remove all stops times not belonging to any known trip
	delStopTimesStmt = `
DELETE
FROM
	stop_times
WHERE trip_id NOT IN (
	SELECT DISTINCT
		id
	FROM
		trips);
`

	// statement to remove stops that don't have a stop time associated
	delStopsStmt = `
DELETE
FROM
	stops
WHERE
	id NOT IN (
	SELECT DISTINCT
		stop_id
	FROM
		stop_times);
`

	// statement to remove all shapes that don't belong to any relevant trip
	delShapesStmt = `
DELETE
FROM
	shapes
WHERE
	shape_id NOT IN (
	SELECT DISTINCT
		shape_id
	FROM
		trips);
`
)

// trimItemsResult is the type used to describe the result of trimming a single item type.
type trimItemsResult struct {
	ItemType  gtfs.ItemType
	Affected  int64
	Remaining int64
	Time      time.Duration
}

// String returns a human-readable representation of trimItemsResult.
func (tir trimItemsResult) String() string {
	return fmt.Sprintf("trimed %d %s to %d in %s", tir.Affected, tir.ItemType, tir.Remaining, tir.Time)
}

// trimResult is the type used to describe the result of trimming all item types.
type trimResult map[gtfs.ItemType]*trimItemsResult

// String returns a human-readable representation of trimResult.
func (tr trimResult) String() string {
	var sb strings.Builder
	for _, trimItemsResult := range tr {
		sb.WriteString(fmt.Sprintf("%s\n", trimItemsResult))
	}
	return sb.String()
}

func gtfsTrim(_ *cobra.Command, args []string) error {
	dbPath := args[0]
	agency := args[1]

	// some argument validation
	if dbPath == "" {
		return errors.New("empty dbPath")
	}
	if agency == "" {
		return errors.New("empty agency")
	}

	// open gorm db
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return err
	}

	// ensure tables matching our model
	err = gtfs.Migrate(db)
	if err != nil {
		return fmt.Errorf("failed to migrate DB: %w", err)
	}

	// trim to agency
	r, errTrim := trim(db, agency)
	if errTrim != nil {
		if errors.Is(errTrim, gorm.ErrRecordNotFound) {
			log.Println(fmt.Sprintf("could not find an agency like '%s', not trimming", agency))
		} else {
			return fmt.Errorf("failed to trim DB: %w", errTrim)
		}
	}
	log.Println(r.String())

	return nil
}

// trim removes all items from the DB that are not associated with the agency
// that matches like. After completion, trim returns some stats.
func trim(db *gorm.DB, like string) (*trimResult, error) {

	// ensure all necessary tables are available for stripping
	requiredTables := []string{"agencies", "routes", "trips", "stop_times", "stops", "shapes", "calendars", "calendar_dates"}
	for _, tableName := range requiredTables {
		if !db.Migrator().HasTable(tableName) {
			return nil, fmt.Errorf("missing table '%s'", tableName)
		}
	}

	var agency gtfs.Agency
	tx := db.Where("name LIKE ?", fmt.Sprintf("%%%s%%", like)).First(&agency)
	if tx.Error != nil {
		return nil, tx.Error
	}

	// trim config (note, the order of executing the trim statements is relevant)
	config := []struct {
		itemType gtfs.ItemType
		stmt     string
		tblName  string
		values   []interface{}
	}{
		{gtfs.Agencies, delAgencyStmt, "agencies", []interface{}{agency.ID}},
		{gtfs.Routes, delRoutesStmt, "routes", nil},
		{gtfs.Trips, delTripsStmt, "trips", nil},
		{gtfs.StopTimes, delStopTimesStmt, "stop_times", nil},
		{gtfs.Stops, delStopsStmt, "stops", nil},
		{gtfs.Shapes, delShapesStmt, "shapes", nil},
		// TODO: also trim calendar and calendar_dates
	}

	// execute each of the statements
	trimResult := trimResult{}
	for _, c := range config {

		start := time.Now()
		tx := db.Exec(c.stmt, c.values...)
		if tx.Error != nil {
			return nil, fmt.Errorf("failed to trim %s: %w", c.itemType, tx.Error)
		}
		trimItemsResult := trimItemsResult{
			ItemType: c.itemType,
			Affected: tx.RowsAffected,
			Time:     time.Since(start),
		}
		db.Table(c.tblName).Count(&trimItemsResult.Remaining)
		trimResult[c.itemType] = &trimItemsResult

	}

	return &trimResult, nil
}
