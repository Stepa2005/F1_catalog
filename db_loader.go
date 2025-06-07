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
	query := `SELECT circuitId, circuitRef, name, location, country, lat, lng, alt, url FROM circuits`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	circuits := make(map[string]Circuit)
	for rows.Next() {
		var c Circuit
		err := rows.Scan(&c.ID, &c.Ref, &c.Name, &c.Location, &c.Country, &c.Lat, &c.Lng, &c.Alt, &c.URL)
		if err != nil {
			log.Printf("Error scanning circuit: %v", err)
			continue
		}
		circuits[c.ID] = c
	}
	return circuits, nil
}

func loadConstructorsFromDB(db *sql.DB) (map[string]Constructor, error) {
	query := `SELECT constructorId, constructorRef, name, nationality, url FROM constructors`
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
	query := `SELECT driverId, driverRef, number, code, forename, surname, dob, nationality, url FROM drivers`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	drivers := make(map[string]Driver)
	for rows.Next() {
		var d Driver
		err := rows.Scan(&d.ID, &d.Ref, &d.Number, &d.Code, &d.Forename, &d.Surname, &d.DOB, &d.Nationality, &d.URL)
		if err != nil {
			log.Printf("Error scanning driver: %v", err)
			continue
		}
		drivers[d.ID] = d
	}
	return drivers, nil
}

func loadRacesFromDB(db *sql.DB) ([]Race, error) {
	query := `SELECT raceId, year, round, circuitId, name, date, time, url, 
		fp1_date, fp1_time, fp2_date, fp2_time, fp3_date, fp3_time, 
		quali_date, quali_time, sprint_date, sprint_time FROM races`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var races []Race
	for rows.Next() {
		var r Race
		err := rows.Scan(&r.ID, &r.Year, &r.Round, &r.CircuitID, &r.Name, &r.Date, &r.Time, &r.URL,
			&r.Fp1Date, &r.Fp1Time, &r.Fp2Date, &r.Fp2Time, &r.Fp3Date, &r.Fp3Time,
			&r.QualiDate, &r.QualiTime, &r.SprintDate, &r.SprintTime)

		if err != nil {
			log.Printf("Error scanning race: %v", err)
			continue
		}
		races = append(races, r)
	}
	return races, nil
}

func loadResultsFromDB(db *sql.DB) ([]Result, error) {
	query := `SELECT resultId, raceId, driverId, constructorId, number, grid, 
		position, positionText, positionOrder, points, laps, time, milliseconds, 
		fastestLap, rank, fastestLapTime, fastestLapSpeed, statusId FROM results`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var res Result
		err := rows.Scan(&res.ID, &res.RaceID, &res.DriverID, &res.ConstructorID, &res.Number, &res.Grid,
			&res.Position, &res.PositionText, &res.PositionOrder, &res.Points, &res.Laps, &res.Time, &res.Milliseconds,
			&res.FastestLap, &res.Rank, &res.FastestLapTime, &res.FastestLapSpeed, &res.StatusID)

		if err != nil {
			log.Printf("Error scanning result: %v", err)
			continue
		}
		results = append(results, res)
	}
	return results, nil
}

func loadStatusesFromDB(db *sql.DB) (map[string]Status, error) {
	query := `SELECT statusId, status FROM status`
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
	query := `SELECT year, url FROM seasons`
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