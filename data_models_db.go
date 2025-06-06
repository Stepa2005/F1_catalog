package main

import (
	"database/sql"
	"time"
)

type Circuit struct {
	ID       string
	Ref      string
	Name     string
	Location string
	Country  string
	Lat      string
	Lng      string
	Alt      sql.NullString
	URL      string
}

type Constructor struct {
	ID          string
	Ref         string
	Name        string
	Nationality string
	URL         string
}

type Driver struct {
	ID          string
	Ref         string
	Number      sql.NullInt32
	Code        sql.NullString
	Forename    string
	Surname     string
	DOB         time.Time
	Nationality string
	URL         string
}

type Race struct {
	ID         string
	Year       int
	Round      int
	CircuitID  string
	Name       string
	Date       time.Time
	Time       sql.NullString
	URL        sql.NullString
	Fp1Date    sql.NullTime
	Fp1Time    sql.NullString
	Fp2Date    sql.NullTime
	Fp2Time    sql.NullString
	Fp3Date    sql.NullTime
	Fp3Time    sql.NullString
	QualiDate  sql.NullTime
	QualiTime  sql.NullString
	SprintDate sql.NullTime
	SprintTime sql.NullString
}

type Result struct {
	ID              string
	RaceID          string
	DriverID        string
	ConstructorID   string
	Number          sql.NullInt32
	Grid            int
	Position        sql.NullInt32
	PositionText    string
	PositionOrder   int
	Points          float32
	Laps            int
	Time            sql.NullString
	Milliseconds    sql.NullInt32
	FastestLap      sql.NullInt32
	Rank            sql.NullInt32
	FastestLapTime  sql.NullString
	FastestLapSpeed sql.NullString
	StatusID        string
}

type Status struct {
	ID     string
	Status string
}

type Season struct {
	Year int
	URL  string
}

// UI модели данных
type Location struct {
	Lat      string
	Long     string
	Locality string
	Country  string
}

type Session struct {
	Date string
	Time string
}

type RaceResult struct {
	Position    string
	Number      string
	Driver      Driver
	Constructor Constructor
	Grid        string
	Laps        string
	Time        *struct{ Millis, Time string }
	Status      string
	Points      string
}

type RaceUI struct {
	RaceIDInternal string
	Round          string
	RaceName       string
	Date           string
	Circuit        Circuit
	URL            string
	FirstPractice  Session
	SecondPractice Session
	ThirdPractice  *Session
	Qualifying     Session
	Sprint         *Session
}
