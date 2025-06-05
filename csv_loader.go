package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const datasetDir = "dataset" // Убедитесь, что эта папка существует в корне проекта

// loadAllDataFromCSVSync загружает все данные из CSV-файлов.
// Должна вызываться перед первым использованием данных.
func loadAllDataFromCSVSync() error {
	if csvDataLoaded {
		return nil
	}
	fmt.Println("Loading CSV data...")

	var err error

	// Circuits
	allCircuitsData, err = loadCSVToMap(filepath.Join(datasetDir, "circuits.csv"), parseCircuitCSVRecord, func(r CircuitCSVRecord) string { return r.CircuitID })
	if err != nil {
		return fmt.Errorf("failed to load circuits.csv: %w", err)
	}

	// Constructors
	allConstructorsData, err = loadCSVToMap(filepath.Join(datasetDir, "constructors.csv"), parseConstructorCSVRecord, func(r ConstructorCSVRecord) string { return r.ConstructorID })
	if err != nil {
		return fmt.Errorf("failed to load constructors.csv: %w", err)
	}

	// Drivers
	allDriversData, err = loadCSVToMap(filepath.Join(datasetDir, "drivers.csv"), parseDriverCSVRecord, func(r DriverCSVRecord) string { return r.DriverID })
	if err != nil {
		return fmt.Errorf("failed to load drivers.csv: %w", err)
	}

	// Races
	racesSlice, err := loadCSVToSlice(filepath.Join(datasetDir, "races.csv"), parseRaceCSVRecord)
	if err != nil {
		return fmt.Errorf("failed to load races.csv: %w", err)
	}
	allRacesData = racesSlice

	// Results
	resultsSlice, err := loadCSVToSlice(filepath.Join(datasetDir, "results.csv"), parseResultCSVRecord)
	if err != nil {
		return fmt.Errorf("failed to load results.csv: %w", err)
	}
	allResultsData = resultsSlice
    // Сортируем результаты по raceId и затем по positionOrder для более предсказуемого поведения
    sort.SliceStable(allResultsData, func(i, j int) bool {
        if allResultsData[i].RaceID != allResultsData[j].RaceID {
            return allResultsData[i].RaceID < allResultsData[j].RaceID
        }
        posI, _ := strconv.Atoi(allResultsData[i].PositionOrder)
        posJ, _ := strconv.Atoi(allResultsData[j].PositionOrder)
        return posI < posJ
    })


	// Status
	allStatusesData, err = loadCSVToMap(filepath.Join(datasetDir, "status.csv"), parseStatusCSVRecord, func(r StatusCSVRecord) string { return r.StatusID })
	if err != nil {
		return fmt.Errorf("failed to load status.csv: %w", err)
	}

	// Seasons
	allSeasonsData, err = loadCSVToMap(filepath.Join(datasetDir, "seasons.csv"), parseSeasonCSVRecord, func(r SeasonCSVRecord) string { return r.Year })
	if err != nil {
		return fmt.Errorf("failed to load seasons.csv: %w", err)
	}

	csvDataLoaded = true
	fmt.Println("CSV data loaded successfully.")
	return nil
}

// loadCSVToMap - общая функция для загрузки CSV в карту.
func loadCSVToMap[T any, K comparable](filePath string, parserFunc func([]string) (T, error), keyExtractor func(T) K) (map[K]T, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, err = reader.Read() // Пропустить заголовок
	if err != nil {
		return nil, fmt.Errorf("reading header from %s: %w", filePath, err)
	}

	dataMap := make(map[K]T)
	lineNumber := 1
	for {
		lineNumber++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading record from %s at line %d: %w", filePath, lineNumber, err)
		}

		parsedRecord, err := parserFunc(record)
		if err != nil {
			fmt.Printf("Error parsing record in %s at line %d: %v (record: %v)\n", filePath, lineNumber, err, record)
			continue // Пропустить ошибочные записи или вернуть ошибку
		}
		dataMap[keyExtractor(parsedRecord)] = parsedRecord
	}
	return dataMap, nil
}

// loadCSVToSlice - общая функция для загрузки CSV в срез.
func loadCSVToSlice[T any](filePath string, parserFunc func([]string) (T, error)) ([]T, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, err = reader.Read() // Пропустить заголовок
	if err != nil {
		return nil, fmt.Errorf("reading header from %s: %w", filePath, err)
	}

	var dataSlice []T
	lineNumber := 1
	for {
		lineNumber++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading record from %s at line %d: %w", filePath, lineNumber, err)
		}

		parsedRecord, err := parserFunc(record)
		if err != nil {
			fmt.Printf("Error parsing record in %s at line %d: %v (record: %v)\n", filePath, lineNumber, err, record)
			continue
		}
		dataSlice = append(dataSlice, parsedRecord)
	}
	return dataSlice, nil
}

// --- Функции парсинга для каждой CSV-структуры ---

func parseCircuitCSVRecord(r []string) (CircuitCSVRecord, error) {
	if len(r) < 9 {
		return CircuitCSVRecord{}, fmt.Errorf("expected 9 fields, got %d", len(r))
	}
	return CircuitCSVRecord{
		CircuitID: r[0], CircuitRef: r[1], Name: r[2], Location: r[3], Country: r[4],
		Lat: r[5], Lng: r[6], Alt: csvOptionalString(r[7]), URL: r[8],
	}, nil
}

func parseConstructorCSVRecord(r []string) (ConstructorCSVRecord, error) {
	if len(r) < 5 {
		return ConstructorCSVRecord{}, fmt.Errorf("expected 5 fields, got %d", len(r))
	}
	return ConstructorCSVRecord{
		ConstructorID: r[0], ConstructorRef: r[1], Name: r[2], Nationality: r[3], URL: r[4],
	}, nil
}

func parseDriverCSVRecord(r []string) (DriverCSVRecord, error) {
	if len(r) < 9 {
		return DriverCSVRecord{}, fmt.Errorf("expected 9 fields, got %d", len(r))
	}
	return DriverCSVRecord{
		DriverID: r[0], DriverRef: r[1], Number: csvOptionalString(r[2]), Code: csvOptionalString(r[3]),
		Forename: r[4], Surname: r[5], DOB: r[6], Nationality: r[7], URL: r[8],
	}, nil
}

func parseRaceCSVRecord(r []string) (RaceCSVRecord, error) {
	if len(r) < 18 {
		return RaceCSVRecord{}, fmt.Errorf("expected 18 fields, got %d for race record", len(r))
	}
	return RaceCSVRecord{
		RaceID: r[0], Year: r[1], Round: r[2], CircuitID: r[3], Name: r[4], Date: r[5],
		Time: csvOptionalString(r[6]), URL: csvOptionalString(r[7]),
		Fp1Date: csvOptionalString(r[8]), Fp1Time: csvOptionalString(r[9]),
		Fp2Date: csvOptionalString(r[10]), Fp2Time: csvOptionalString(r[11]),
		Fp3Date: csvOptionalString(r[12]), Fp3Time: csvOptionalString(r[13]),
		QualiDate: csvOptionalString(r[14]), QualiTime: csvOptionalString(r[15]),
		SprintDate: csvOptionalString(r[16]), SprintTime: csvOptionalString(r[17]),
	}, nil
}

func parseResultCSVRecord(r []string) (ResultCSVRecord, error) {
	if len(r) < 18 {
		return ResultCSVRecord{}, fmt.Errorf("expected 18 fields, got %d for result record", len(r))
	}
	return ResultCSVRecord{
		ResultID: r[0], RaceID: r[1], DriverID: r[2], ConstructorID: r[3],
		Number: csvOptionalString(r[4]), Grid: r[5], Position: csvOptionalString(r[6]),
		PositionText: r[7], PositionOrder: r[8], Points: r[9], Laps: r[10],
		Time: csvOptionalString(r[11]), Milliseconds: csvOptionalString(r[12]),
		FastestLap: csvOptionalString(r[13]), Rank: csvOptionalString(r[14]),
		FastestLapTime: csvOptionalString(r[15]), FastestLapSpeed: csvOptionalString(r[16]),
		StatusID: r[17],
	}, nil
}

func parseStatusCSVRecord(r []string) (StatusCSVRecord, error) {
	if len(r) < 2 {
		return StatusCSVRecord{}, fmt.Errorf("expected 2 fields, got %d", len(r))
	}
	return StatusCSVRecord{StatusID: r[0], Status: r[1]}, nil
}

func parseSeasonCSVRecord(r []string) (SeasonCSVRecord, error) {
	if len(r) < 2 {
		return SeasonCSVRecord{}, fmt.Errorf("expected 2 fields, got %d", len(r))
	}
	return SeasonCSVRecord{Year: r[0], URL: r[1]}, nil
}