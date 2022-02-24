package gtfs

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"math"
	"strconv"
	"strings"
)

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
