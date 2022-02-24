package gtfs

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/gocarina/gocsv"
	"gorm.io/gorm"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// batchSize is the size of the batches to use for importing into the DB.
const batchSize = 1000

type DateTime struct {
	int32
}

// MarshalCSV marshals DateTime to CSV (i.e. when writing to CSV).
func (dt *DateTime) MarshalCSV() (string, error) {

	hours := dt.int32 / 3600
	minutes := (dt.int32 % 3600) / 60
	seconds := (dt.int32 % 3600) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds), nil
}

// UnmarshalCSV unmarshalls CSV to DateTime (i.e. when reading from CSV).
func (dt *DateTime) UnmarshalCSV(csv string) error {
	s := strings.Split(csv, ":")
	if len(s) != 3 {
		return errors.New(fmt.Sprintf("cannot parse GTFS Time from '%s'", csv))
	}
	hours, err := strconv.Atoi(s[0])
	if err != nil {
		return fmt.Errorf("cannot parse GTFS hours from '%s': %w", s[0], err)
	}
	minutes, err := strconv.Atoi(s[1])
	if err != nil {
		return fmt.Errorf("cannot parse GTFS minutes from '%s': %w", s[0], err)
	}
	seconds, err := strconv.Atoi(s[2])
	if err != nil {
		return fmt.Errorf("cannot parse GTFS seconds from '%s': %w", s[0], err)
	}
	i := int64(hours*3600 + minutes*60 + seconds)
	if i > math.MaxInt32 {
		return fmt.Errorf("cannot parse GTFS time from '%s': max value exceeded", csv)
	}
	dt.int32 = int32(i)
	return nil
}

// Scan converts from DB to DateTime.
func (dt *DateTime) Scan(value interface{}) error {
	i, ok := value.(int64)
	if !ok {
		return fmt.Errorf("cannot scan '%v' to GTFS Time", value)
	}
	if i > math.MaxInt32 {
		return fmt.Errorf("cannot scan '%v' to GTFS Tim: max value exceeded", value)
	}
	dt.int32 = int32(i)
	return nil
}

// Value converts from DateTime to DB.
func (dt DateTime) Value() (driver.Value, error) {
	return int64(dt.int32), nil
}

// Agency model.
type Agency struct {
	ID   string `csv:"agency_id"`
	Name string `csv:"agency_name"`
	URL  string `csv:"agency_url"`
	//Timezone string `csv:"agency_timezone"`
	//Language string `csv:"agency_lang"`
	//Phone    string `csv:"agency_phone"`
}

// Route model.
type Route struct {
	ID        string `csv:"route_id"`
	AgencyID  string `csv:"agency_id"`
	Agency    Agency
	ShortName string `csv:"route_short_name"`
	LongName  string `csv:"route_long_name"`
	Type      int    `csv:"route_type"`
	//Desc      string `csv:"route_url"`
	//URL       string `csv:"route_desc"`
	//Color     string `csv:"route_color"`
	//TextColor string `csv:"route_text_color"`
}

// Trip model.
type Trip struct {
	ID          string `csv:"trip_id"`
	Name        string `csv:"trip_short_name"`
	RouteID     string `csv:"route_id"`
	Route       Route
	ServiceID   string `csv:"service_id"`
	DirectionID string `csv:"direction_id"`
	ShapeID     string `csv:"shape_id"`
	//ServiceID   string `csv:"service_id"`
}

// StopTime model.
type StopTime struct {
	ID        uint   `gorm:"primaryKey,autoIncrement"`
	StopID    string `csv:"stop_id"`
	Stop      Stop
	TripID    string `csv:"trip_id"`
	Trip      Trip
	Departure DateTime `csv:"departure_time"`
	Arrival   DateTime `csv:"arrival_time"`
	StopSeq   int      `csv:"stop_sequence"`
	//StopHeadSign string `csv:"stop_headsign"`
	//Shape        float64 `csv:"shape_dist_traveled"`
}

// Stop model.
type Stop struct {
	ID        string  `csv:"stop_id"`
	Name      string  `csv:"stop_name"`
	Latitude  float64 `csv:"stop_lat"`
	Longitude float64 `csv:"stop_lon"`
	// Code        string  `csv:"stop_code"`
	// Description string  `csv:"stop_desc"`
	// Type        string  `csv:"location_type"`
	// Parent      string  `csv:"parent_station"`
}

// Shape model.
type Shape struct {
	ID         uint    `gorm:"primaryKey,autoIncrement"`
	ShapeID    string  `csv:"shape_id"`
	PtLat      float64 `csv:"shape_pt_lat"`
	PtLon      float64 `csv:"shape_pt_lon"`
	PtSequence int     `csv:"shape_pt_sequence"`
}

// Calendar model.
type Calendar struct {
	ID        uint   `gorm:"primaryKey,autoIncrement"`
	ServiceID string `csv:"service_id"`
	Monday    int    `csv:"monday"`
	Tuesday   int    `csv:"tuesday"`
	Wednesday int    `csv:"wednesday"`
	Thursday  int    `csv:"thursday"`
	Friday    int    `csv:"friday"`
	Saturday  int    `csv:"saturday"`
	Sunday    int    `csv:"sunday"`
	StartDate string `csv:"start_date"`
	EndDate   string `csv:"end_date"`
}

// CalendarDate model.
type CalendarDate struct {
	ID            uint   `gorm:"primaryKey,autoIncrement"`
	ServiceID     string `csv:"service_id"`
	Date          string `csv:"date"`
	ExceptionType int    `csv:"exception_type"`
}

type RouteStop struct {
	StopID    string
	RouteID   string
	Direction string
}

// ItemType enumerates different item types.
type ItemType uint32

const (

	// Agencies the item type for agency items.
	Agencies ItemType = iota

	// Routes the item type for route items.
	Routes

	// Trips the item type for trip items.
	Trips

	// Stops the item type for stop items.
	Stops

	// StopTimes the item type for stop time items.
	StopTimes

	// Shapes the item type for shape items.
	Shapes

	// Calendars the item type for shape items.
	Calendars

	// CalendarDates the item type for shape items.
	CalendarDates
)

var txItemType = map[ItemType]string{
	Agencies:      "Agencies",
	Routes:        "Routes",
	Trips:         "Trips",
	Stops:         "Stops",
	StopTimes:     "Stop Times",
	Shapes:        "Shapes",
	Calendars:     "Calendars",
	CalendarDates: "Calendar Dates",
}

// String returns a human-readable representation of ItemType.
func (it ItemType) String() string {
	if s := txItemType[it]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown Status (%d)", uint32(it))
}

// ImportItemsResult is the type used to describe the result of importing a single item type.
type ImportItemsResult struct {
	ItemType ItemType
	Count    int64
	Batches  int64
	Time     time.Duration
	Error    error
}

// String returns a human-readable representation of ImportItemsResult.
func (iir ImportItemsResult) String() string {
	if iir.Error != nil {
		return fmt.Sprintf("failed to import %s: %v", iir.ItemType, iir.Error)
	}
	return fmt.Sprintf("imported %d %s in %d batches in %s", iir.Count, iir.ItemType, iir.Batches, iir.Time)
}

// Import GTFS CSV files from the directory gtfsBase into the db.
//
// If the progress channel is not nil, import results (for each of the item
// types) will be sent through the channel.
func Import(db *gorm.DB, gtfsBase string, progress chan *ImportItemsResult) {

	// define what to import
	sources := []struct {
		path     string
		itemType ItemType
	}{
		{path.Join(gtfsBase, "agency.txt"), Agencies},
		{path.Join(gtfsBase, "routes.txt"), Routes},
		{path.Join(gtfsBase, "trips.txt"), Trips},
		{path.Join(gtfsBase, "stops.txt"), Stops},
		{path.Join(gtfsBase, "stop_times.txt"), StopTimes},
		{path.Join(gtfsBase, "shapes.txt"), Shapes},
		{path.Join(gtfsBase, "calendar.txt"), Calendars},
		{path.Join(gtfsBase, "calendar_dates.txt"), CalendarDates},
	}

	// import each of the sources
	for _, source := range sources {
		importItemsResult := importItems(source.path, db, source.itemType)

		// send progress if desired
		if progress != nil {
			progress <- importItemsResult
		}
	}

	if progress != nil {
		close(progress)
	}
}

// Migrate ensure the given DB matches our models.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Agency{},
		&Route{},
		&Trip{},
		&StopTime{},
		&Stop{},
		&Shape{},
		&Calendar{},
		&CalendarDate{},
	)
}

// importItems imports all items of a given type from a CSV-file into a DB.
func importItems(csvPath string, db *gorm.DB, importType ItemType) *ImportItemsResult {

	// provide for timing
	start := time.Now()

	// parse CSV and send each row to the channel (UnmarshalToChan closes the channel)
	file, err := os.Open(csvPath)
	if err != nil {
		return &ImportItemsResult{Error: err}
	}
	defer func() {
		_ = file.Close()
	}()

	resultChan := make(chan *ImportItemsResult)

	var itemChan interface{}
	switch importType {
	case Agencies:
		c := make(chan *Agency)
		go batchImportAgencies(c, resultChan, db)
		itemChan = c
	case Routes:
		c := make(chan *Route)
		go batchImportRoutes(c, resultChan, db)
		itemChan = c
	case Trips:
		c := make(chan *Trip)
		go batchImportTrips(c, resultChan, db)
		itemChan = c
	case Stops:
		c := make(chan *Stop)
		go batchImportStops(c, resultChan, db)
		itemChan = c
	case StopTimes:
		c := make(chan *StopTime)
		go batchImportStopTimes(c, resultChan, db)
		itemChan = c
	case Shapes:
		c := make(chan *Shape)
		go batchImportShapes(c, resultChan, db)
		itemChan = c
	case Calendars:
		c := make(chan *Calendar)
		go batchImportCalendars(c, resultChan, db)
		itemChan = c
	case CalendarDates:
		c := make(chan *CalendarDate)
		go batchImportCalendarDates(c, resultChan, db)
		itemChan = c
	default:
		return &ImportItemsResult{Error: fmt.Errorf("unknown ItemType %d", importType)}
	}

	if err = gocsv.UnmarshalToChan(file, itemChan); err != nil {
		return &ImportItemsResult{Error: err}
	}

	// wait for the batch insert to return counts
	r := <-resultChan

	// compute the elapsed Time
	r.Time = time.Now().Sub(start)

	return r
}

// batchImportShapes imports all shapes from a channel into a DB.
func batchImportAgencies(items chan *Agency, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*Agency

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
				result <- &ImportItemsResult{ItemType: Agencies, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*Agency{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: Agencies, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: Agencies, Count: itemCount, Batches: batchCount}
}

// batchImportRoutes imports all routes from a channel into a DB.
func batchImportRoutes(items chan *Route, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*Route

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
				result <- &ImportItemsResult{ItemType: Routes, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*Route{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: Routes, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: Routes, Count: itemCount, Batches: batchCount}
}

// batchImportTrips imports all trips from a channel into a DB.
func batchImportTrips(items chan *Trip, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*Trip

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
				result <- &ImportItemsResult{ItemType: Trips, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*Trip{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: Trips, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: Trips, Count: itemCount, Batches: batchCount}
}

// batchImportStops imports all stops from a channel into a DB.
func batchImportStops(items chan *Stop, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*Stop

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
				result <- &ImportItemsResult{ItemType: Stops, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*Stop{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: Stops, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: Stops, Count: itemCount, Batches: batchCount}
}

// batchImportStopTimes imports all stopTimes from a channel into a DB.
func batchImportStopTimes(items chan *StopTime, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*StopTime

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
				result <- &ImportItemsResult{ItemType: StopTimes, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*StopTime{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: StopTimes, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: StopTimes, Count: itemCount, Batches: batchCount}
}

// batchImportShapes imports all shapes from a channel into a DB.
func batchImportShapes(items chan *Shape, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*Shape

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
				result <- &ImportItemsResult{ItemType: Shapes, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*Shape{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: Shapes, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: Shapes, Count: itemCount, Batches: batchCount}
}

// batchImportCalendars imports all calendars from a channel into a DB.
func batchImportCalendars(items chan *Calendar, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*Calendar

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
				result <- &ImportItemsResult{ItemType: Calendars, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*Calendar{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: Calendars, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: Calendars, Count: itemCount, Batches: batchCount}
}

// batchImportCalendarDates imports all calendars from a channel into a DB.
func batchImportCalendarDates(items chan *CalendarDate, result chan *ImportItemsResult, db *gorm.DB) {

	// ensure the result channel will be closed at last
	defer close(result)

	// initialize counters
	var itemCount int64
	var batchCount int64

	// initialize the batch
	var batch []*CalendarDate

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
				result <- &ImportItemsResult{ItemType: CalendarDates, Error: tx.Error}
				return
			}
			batchCount++

			// reset batch
			batch = []*CalendarDate{}
		}
	}

	// persist any incomplete batch
	if len(batch) > 0 {
		tx := db.Create(batch)
		if tx.Error != nil {
			result <- &ImportItemsResult{ItemType: CalendarDates, Error: tx.Error}
			return
		}
		batchCount++
	}

	// return the counts
	result <- &ImportItemsResult{ItemType: CalendarDates, Count: itemCount, Batches: batchCount}
}
