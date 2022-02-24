package commands

import (
	"errors"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"heimdalr/gtfs"
	"log"
	"os"
	"path"
	"time"
)

// batchSize is the size of the batches to use for importing into the DB.
const batchSize = 1000

// importResult is the type used to describe the result of importing a single item type.
type importResult struct {
	ItemType gtfs.ItemType
	Count    int64
	Batches  int64
	Time     time.Duration
	Error    error
}

// String returns a human-readable representation of importResult.
func (iir importResult) String() string {
	if iir.Error != nil {
		return fmt.Sprintf("failed to import %s: %v", iir.ItemType, iir.Error)
	}
	return fmt.Sprintf("imported %d %s in %d batches in %s", iir.Count, iir.ItemType, iir.Batches, iir.Time)
}

func gtfsImport(_ *cobra.Command, args []string) error {

	gtfsBasePath := args[0]
	dbPath := args[1]

	// some argument validation
	if gtfsBasePath == "" {
		return errors.New("empty gtfsBasePath")
	}
	if dbPath == "" {
		return errors.New("empty dbPath")
	}

	// delete db-file, if it exists
	_, err := os.Stat(dbPath)
	if err == nil {
		if err = os.Remove(dbPath); err != nil {
			return fmt.Errorf("failed to remove old db file '%s'", dbPath)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// open gorm db
	var db *gorm.DB
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
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

	// import CSV files
	progress := make(chan *importResult)
	go importAll(db, gtfsBasePath, progress)
	for r := range progress {
		log.Println(r.String())
	}

	return nil
}

// importAll imports all GTFS CSV files from the directory gtfsBase into the
// db.
//
// If the progress channel is not nil, import results (for each of the item
// types) will be sent through the channel.
func importAll(db *gorm.DB, gtfsBase string, progress chan *importResult) {

	// define what to import
	sources := []struct {
		path     string
		itemType gtfs.ItemType
	}{
		{path.Join(gtfsBase, "agency.txt"), gtfs.Agencies},
		{path.Join(gtfsBase, "routes.txt"), gtfs.Routes},
		{path.Join(gtfsBase, "trips.txt"), gtfs.Trips},
		{path.Join(gtfsBase, "stops.txt"), gtfs.Stops},
		{path.Join(gtfsBase, "stop_times.txt"), gtfs.StopTimes},
		{path.Join(gtfsBase, "shapes.txt"), gtfs.Shapes},
		{path.Join(gtfsBase, "calendar.txt"), gtfs.Calendars},
		{path.Join(gtfsBase, "calendar_dates.txt"), gtfs.CalendarDates},
	}

	// import each of the sources
	for _, source := range sources {
		r := importSingle(source.path, db, source.itemType)

		// send progress if desired
		if progress != nil {
			progress <- r
		}
	}

	if progress != nil {
		close(progress)
	}
}

// importSingle imports all items of a given type from a CSV-file into a DB.
func importSingle(csvPath string, db *gorm.DB, importType gtfs.ItemType) *importResult {

	// provide for timing
	start := time.Now()

	// parse CSV and send each row to the channel (UnmarshalToChan closes the channel)
	file, err := os.Open(csvPath)
	if err != nil {
		return &importResult{Error: err}
	}
	defer func() {
		_ = file.Close()
	}()

	resultChan := make(chan *importResult)

	var itemChan interface{}
	switch importType {
	case gtfs.Agencies:
		c := make(chan *gtfs.Agency)
		go importAgencies(c, resultChan, db)
		itemChan = c
	case gtfs.Routes:
		c := make(chan *gtfs.Route)
		go importRoutes(c, resultChan, db)
		itemChan = c
	case gtfs.Trips:
		c := make(chan *gtfs.Trip)
		go importTrips(c, resultChan, db)
		itemChan = c
	case gtfs.Stops:
		c := make(chan *gtfs.Stop)
		go importStops(c, resultChan, db)
		itemChan = c
	case gtfs.StopTimes:
		c := make(chan *gtfs.StopTime)
		go importStopTimes(c, resultChan, db)
		itemChan = c
	case gtfs.Shapes:
		c := make(chan *gtfs.Shape)
		go importShapes(c, resultChan, db)
		itemChan = c
	case gtfs.Calendars:
		c := make(chan *gtfs.Calendar)
		go importCalendars(c, resultChan, db)
		itemChan = c
	case gtfs.CalendarDates:
		c := make(chan *gtfs.CalendarDate)
		go importCalendarDates(c, resultChan, db)
		itemChan = c
	default:
		return &importResult{Error: fmt.Errorf("unknown ItemType %d", importType)}
	}

	if err = gocsv.UnmarshalToChan(file, itemChan); err != nil {
		return &importResult{Error: err}
	}

	// wait for the batch insert to return counts
	r := <-resultChan

	// compute the elapsed Time
	r.Time = time.Since(start)

	return r
}

// importShapes imports all shapes from a channel into a DB.
func importAgencies(items chan *gtfs.Agency, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.Agency

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.Agencies, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.Agency{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.Agencies, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.Agencies, Count: itemCount, Batches: batchCount}
}

// importRoutes imports all routes from a channel into a DB.
func importRoutes(items chan *gtfs.Route, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.Route

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.Routes, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.Route{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.Routes, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.Routes, Count: itemCount, Batches: batchCount}
}

// importTrips imports all trips from a channel into a DB.
func importTrips(items chan *gtfs.Trip, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.Trip

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.Trips, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.Trip{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.Trips, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.Trips, Count: itemCount, Batches: batchCount}
}

// importStops imports all stops from a channel into a DB.
func importStops(items chan *gtfs.Stop, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.Stop

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.Stops, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.Stop{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.Stops, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.Stops, Count: itemCount, Batches: batchCount}
}

// importStopTimes imports all stopTimes from a channel into a DB.
func importStopTimes(items chan *gtfs.StopTime, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.StopTime

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.StopTimes, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.StopTime{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.StopTimes, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.StopTimes, Count: itemCount, Batches: batchCount}
}

// importShapes imports all shapes from a channel into a DB.
func importShapes(items chan *gtfs.Shape, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.Shape

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.Shapes, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.Shape{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.Shapes, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.Shapes, Count: itemCount, Batches: batchCount}
}

// importCalendars imports all calendars from a channel into a DB.
func importCalendars(items chan *gtfs.Calendar, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.Calendar

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.Calendars, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.Calendar{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.Calendars, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.Calendars, Count: itemCount, Batches: batchCount}
}

// importCalendarDates imports all calendars from a channel into a DB.
func importCalendarDates(items chan *gtfs.CalendarDate, result chan *importResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*gtfs.CalendarDate

	// successively read all items from the channel
	for item := range items {

		// add item to batch and Count it
		itemCount++
		batch = append(batch, item)

		// if batch is "full"
		if len(batch) == batchSize {

			// persist the batch and Count
			tx := db.Create(batch)
			if tx.Error != nil {
				result <- &importResult{ItemType: gtfs.CalendarDates, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*gtfs.CalendarDate{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &importResult{ItemType: gtfs.CalendarDates, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &importResult{ItemType: gtfs.CalendarDates, Count: itemCount, Batches: batchCount}
}
