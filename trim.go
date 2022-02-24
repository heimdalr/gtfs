package gtfs

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
)

// TrimItemsResult is the type used to describe the result of trimming a single item type.
type TrimItemsResult struct {
	ItemType  ItemType
	Affected  int64
	Remaining int64
	Time      time.Duration
}

// String returns a human-readable representation of TrimItemsResult.
func (tir TrimItemsResult) String() string {
	return fmt.Sprintf("trimed %d %s to %d in %s", tir.Affected, tir.ItemType, tir.Remaining, tir.Time)
}

// TrimResult is the type used to describe the result of trimming all item types.
type TrimResult map[ItemType]*TrimItemsResult

// String returns a human-readable representation of TrimResult.
func (tr TrimResult) String() string {
	var sb strings.Builder
	for _, trimItemsResult := range tr {
		sb.WriteString(fmt.Sprintf("%s\n", trimItemsResult))
	}
	return sb.String()
}

// Trim removes all items from the DB that are not associated with the agency
// that matches like. After completion, Trim returns some stats.
func Trim(db *gorm.DB, like string) (*TrimResult, error) {

	// ensure all necessary tables are available for stripping
	requiredTables := []string{"agencies", "routes", "trips", "stop_times", "stops", "shapes", "calendars", "calendar_dates"}
	for _, tableName := range requiredTables {
		if !db.Migrator().HasTable(tableName) {
			return nil, fmt.Errorf("missing table '%s'", tableName)
		}
	}

	var agency Agency
	tx := db.Where("name LIKE ?", fmt.Sprintf("%%%s%%", like)).First(&agency)
	if tx.Error != nil {
		return nil, tx.Error
	}

	// trim config (note, the order of executing the trim statements is relevant)
	config := []struct {
		itemType ItemType
		stmt     string
		tblName  string
		values   []interface{}
	}{
		{Agencies, delAgencyStmt, "agencies", []interface{}{agency.ID}},
		{Routes, delRoutesStmt, "routes", nil},
		{Trips, delTripsStmt, "trips", nil},
		{StopTimes, delStopTimesStmt, "stop_times", nil},
		{Stops, delStopsStmt, "stops", nil},
		{Shapes, delShapesStmt, "shapes", nil},
		// TODO: also trim calendar and calendar_dates
	}

	// execute each of the statements
	trimResult := TrimResult{}
	for _, c := range config {

		start := time.Now()
		tx := db.Exec(c.stmt, c.values...)
		if tx.Error != nil {
			return nil, fmt.Errorf("failed to trim %s: %w", c.itemType, tx.Error)
		}
		trimItemsResult := TrimItemsResult{
			ItemType: c.itemType,
			Affected: tx.RowsAffected,
			Time:     time.Now().Sub(start),
		}
		db.Table(c.tblName).Count(&trimItemsResult.Remaining)
		trimResult[c.itemType] = &trimItemsResult

	}

	return &trimResult, nil
}

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
