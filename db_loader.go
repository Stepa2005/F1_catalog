package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
)

var (
	allCircuitsData     map[string]Circuit
	allConstructorsData map[string]Constructor
	allDriversData      map[string]Driver
	allRacesData        []Race
	allResultsData      []Result
	allStatusesData     map[string]Status
	allSeasonsData      map[string]Season
	dbDataLoaded        bool
)

func loadAllDataFromDBSync(db *sql.DB) error {
	if dbDataLoaded {
		return nil
	}

	fmt.Println("Loading data from database...")

	var err error
	allCircuitsData, err = loadCircuitsFromDB(db)
	if err != nil {
		return fmt.Errorf("failed to load circuits: %w", err)
	}

	allConstructorsData, err = loadConstructorsFromDB(db)
	if err != nil {
		return fmt.Errorf("failed to load constructors: %w", err)
	}

	allDriversData, err = loadDriversFromDB(db)
	if err != nil {
		return fmt.Errorf("failed to load drivers: %w", err)
	}

	allRacesData, err = loadRacesFromDB(db)
	if err != nil {
		return fmt.Errorf("failed to load races: %w", err)
	}

	allResultsData, err = loadResultsFromDB(db)
	if err != nil {
		return fmt.Errorf("failed to load results: %w", err)
	}

	allStatusesData, err = loadStatusesFromDB(db)
	if err != nil {
		return fmt.Errorf("failed to load status: %w", err)
	}

	allSeasonsData, err = loadSeasonsFromDB(db)
	if err != nil {
		return fmt.Errorf("failed to load seasons: %w", err)
	}

	sort.SliceStable(allResultsData, func(i, j int) bool {
		if allResultsData[i].RaceID != allResultsData[j].RaceID {
			return allResultsData[i].RaceID < allResultsData[j].RaceID
		}
		return allResultsData[i].PositionOrder < allResultsData[j].PositionOrder
	})

	dbDataLoaded = true
	fmt.Println("Database data loaded successfully.")
	return nil
}

func loadCircuitsFromDB(db *sql.DB) (map[string]Circuit, error) {
	query := `SELECT circuit_id, circuit_ref, name, location, country, lat, lng, alt, url FROM f1.circuits`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	circuits := make(map[string]Circuit)
	for rows.Next() {
		var c Circuit
		var alt sql.NullString
		err := rows.Scan(&c.ID, &c.Ref, &c.Name, &c.Location, &c.Country, &c.Lat, &c.Lng, &alt, &c.URL)
		if err != nil {
			log.Printf("Error scanning circuit: %v", err)
			continue
		}
		c.Alt = alt
		circuits[c.ID] = c
	}
	return circuits, nil
}

func loadConstructorsFromDB(db *sql.DB) (map[string]Constructor, error) {
	query := `SELECT constructor_id, constructor_ref, name, nationality, url FROM f1.constructors`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	constructors := make(map[string]Constructor)
	for rows.Next() {
		var c Constructor
		err := rows.Scan(&c.ID, &c.Ref, &c.Name, &c.Nationality, &c.URL)
		if err != nil {
			log.Printf("Error scanning constructor: %v", err)
			continue
		}
		constructors[c.ID] = c
	}
	return constructors, nil
}

func loadDriversFromDB(db *sql.DB) (map[string]Driver, error) {
	query := `SELECT driver_id, driver_ref, number, code, forename, surname, dob, nationality, url FROM f1.drivers`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	drivers := make(map[string]Driver)
	for rows.Next() {
		var d Driver
		var number sql.NullInt32
		var code sql.NullString
		err := rows.Scan(&d.ID, &d.Ref, &number, &code, &d.Forename, &d.Surname, &d.DOB, &d.Nationality, &d.URL)
		if err != nil {
			log.Printf("Error scanning driver: %v", err)
			continue
		}
		d.Number = number
		d.Code = code
		drivers[d.ID] = d
	}
	return drivers, nil
}

func loadRacesFromDB(db *sql.DB) ([]Race, error) {
	query := `SELECT race_id, year, round, circuit_id, name, date, time, url, 
		fp1_date, fp1_time, fp2_date, fp2_time, fp3_date, fp3_time, 
		quali_date, quali_time, sprint_date, sprint_time FROM f1.races`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var races []Race
	for rows.Next() {
		var r Race
		var time, url sql.NullString
		var fp1Date, fp2Date, fp3Date, qualiDate, sprintDate sql.NullTime
		var fp1Time, fp2Time, fp3Time, qualiTime, sprintTime sql.NullString

		err := rows.Scan(&r.ID, &r.Year, &r.Round, &r.CircuitID, &r.Name, &r.Date, &time, &url,
			&fp1Date, &fp1Time, &fp2Date, &fp2Time, &fp3Date, &fp3Time,
			&qualiDate, &qualiTime, &sprintDate, &sprintTime)

		if err != nil {
			log.Printf("Error scanning race: %v", err)
			continue
		}

		r.Time = time
		r.URL = url
		r.Fp1Date = fp1Date
		r.Fp1Time = fp1Time
		r.Fp2Date = fp2Date
		r.Fp2Time = fp2Time
		r.Fp3Date = fp3Date
		r.Fp3Time = fp3Time
		r.QualiDate = qualiDate
		r.QualiTime = qualiTime
		r.SprintDate = sprintDate
		r.SprintTime = sprintTime

		races = append(races, r)
	}
	return races, nil
}

func loadResultsFromDB(db *sql.DB) ([]Result, error) {
	query := `SELECT result_id, race_id, driver_id, constructor_id, number, grid, 
		position, position_text, position_order, points, laps, time, milliseconds, 
		fastest_lap, rank, fastest_lap_time, fastest_lap_speed, status_id FROM f1.results`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var res Result
		var number, position, fastestLap, rank sql.NullInt32
		var time, fastestLapTime, fastestLapSpeed sql.NullString
		var milliseconds sql.NullInt32

		err := rows.Scan(&res.ID, &res.RaceID, &res.DriverID, &res.ConstructorID, &number, &res.Grid,
			&position, &res.PositionText, &res.PositionOrder, &res.Points, &res.Laps, &time, &milliseconds,
			&fastestLap, &rank, &fastestLapTime, &fastestLapSpeed, &res.StatusID)

		if err != nil {
			log.Printf("Error scanning result: %v", err)
			continue
		}

		res.Number = number
		res.Position = position
		res.Time = time
		res.Milliseconds = milliseconds
		res.FastestLap = fastestLap
		res.Rank = rank
		res.FastestLapTime = fastestLapTime
		res.FastestLapSpeed = fastestLapSpeed

		results = append(results, res)
	}
	return results, nil
}

func loadStatusesFromDB(db *sql.DB) (map[string]Status, error) {
	query := `SELECT status_id, status FROM f1.status`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statuses := make(map[string]Status)
	for rows.Next() {
		var s Status
		err := rows.Scan(&s.ID, &s.Status)
		if err != nil {
			log.Printf("Error scanning status: %v", err)
			continue
		}
		statuses[s.ID] = s
	}
	return statuses, nil
}

func loadSeasonsFromDB(db *sql.DB) (map[string]Season, error) {
	query := `SELECT year, url FROM f1.seasons`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seasons := make(map[string]Season)
	for rows.Next() {
		var s Season
		err := rows.Scan(&s.Year, &s.URL)
		if err != nil {
			log.Printf("Error scanning season: %v", err)
			continue
		}
		seasons[strconv.Itoa(s.Year)] = s
	}
	return seasons, nil
}
