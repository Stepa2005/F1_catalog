package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	_ "github.com/mattn/go-sqlite3" 
)
// Глобальные переменные UI (без изменений)
var (
	window fyne.Window
	tabs   *container.AppTabs

	racesTabItem *container.TabItem

	resultsTable      *widget.Table
	driversTable      *widget.Table
	constructorsTable *widget.Table

	numColsResults      int
	numColsDrivers      int
	numColsConstructors int

	seasonEntryForInputScreen *widget.Entry
	statusTextForInputScreen  binding.String
	statusLabelForInputScreen *widget.Label

	seasonEntryForDataView *widget.Entry
	statusTextForDataView  binding.String
	statusLabelForDataView *widget.Label

	inputScreen    fyne.CanvasObject
	dataViewScreen fyne.CanvasObject
)

var db *sql.DB
const dbFileName = "f1_data.db" 

func main() {
	myApp := app.New()

	var err error
	db, err = initDB(dbFileName)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	err = loadAllDataFromDBSync(db)
	if err != nil {
		fmt.Printf("CRITICAL: Failed to load initial data: %v\n", err)
	}

	window = myApp.NewWindow("F1 Race Catalog (Database)")
	window.Resize(fyne.NewSize(1200, 700))

	seasonEntryForInputScreen = widget.NewEntry()
	seasonEntryForInputScreen.SetPlaceHolder("Enter season year (e.g., 2023)")

	statusTextForInputScreen = binding.NewString()
	statusLabelForInputScreen = widget.NewLabelWithData(statusTextForInputScreen)
	statusLabelForInputScreen.Wrapping = fyne.TextWrapWord

	loadButtonForInputScreen := widget.NewButtonWithIcon("Load Data", theme.SearchIcon(), func() {
		loadDataForYear(seasonEntryForInputScreen.Text, statusTextForInputScreen, true)
	})
	seasonEntryForInputScreen.OnSubmitted = func(_ string) { loadButtonForInputScreen.OnTapped() }

	wrappedEntry := container.NewGridWrap(fyne.NewSize(300, seasonEntryForInputScreen.MinSize().Height), seasonEntryForInputScreen)
	inputContainer := container.NewVBox(
		widget.NewLabel("Enter Formula 1 Season Year"),
		wrappedEntry,
		loadButtonForInputScreen,
		statusLabelForInputScreen,
	)
	inputScreen = container.NewCenter(inputContainer)

	seasonEntryForDataView = widget.NewEntry()
	seasonEntryForDataView.SetPlaceHolder("Enter another year...")

	statusTextForDataView = binding.NewString()
	statusLabelForDataView = widget.NewLabelWithData(statusTextForDataView)
	statusLabelForDataView.Wrapping = fyne.TextWrapWord

	loadButtonForDataView := widget.NewButtonWithIcon("Load New Season", theme.SearchIcon(), func() {
		loadDataForYear(seasonEntryForDataView.Text, statusTextForDataView, false)
	})
	seasonEntryForDataView.OnSubmitted = func(_ string) { loadButtonForDataView.OnTapped() }

	searchBarForDataView := container.NewBorder(nil, nil, nil, loadButtonForDataView, seasonEntryForDataView)
	topPanelForDataView := container.NewVBox(searchBarForDataView, statusLabelForDataView)

	resultsTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)
	driversTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)
	constructorsTable = widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)

	tabs = container.NewAppTabs()
	initialRacesContent := container.NewCenter(widget.NewLabel("Data will appear here after loading a season."))
	racesTabItem = container.NewTabItem("Races Calendar", initialRacesContent)
	tabs.Append(racesTabItem)
	tabs.Append(container.NewTabItem("Race Results", container.NewVScroll(resultsTable)))
	tabs.Append(container.NewTabItem("Drivers", container.NewVScroll(driversTable)))
	tabs.Append(container.NewTabItem("Constructors", container.NewVScroll(constructorsTable)))

	tabs.OnSelected = func(tabItem *container.TabItem) {
		go func() {
			time.Sleep(100 * time.Millisecond)
			resizeAllVisibleTables()
		}()
	}

	dataViewScreen = container.NewBorder(topPanelForDataView, nil, nil, nil, tabs)

	window.SetContent(inputScreen)
	window.ShowAndRun()
}

// initDB проверяет, существует ли файл БД. Если нет, создает его, применяет схему и заполняет данными из CSV.
func initDB(dbPath string) (*sql.DB, error) {
	_, err := os.Stat(dbPath)
	dbExists := !os.IsNotExist(err)

	// Включаем поддержку foreign keys для SQLite
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if !dbExists {
		fmt.Println("Database file not found. Creating and populating...")

		// Создание таблиц по схеме
		schemaSQL, err := os.ReadFile("schema.sql")
		if err != nil {
			return nil, fmt.Errorf("failed to read schema.sql: %w", err)
		}
		_, err = db.Exec(string(schemaSQL))
		if err != nil {
			return nil, fmt.Errorf("failed to execute schema: %w", err)
		}
		fmt.Println("Tables created successfully.")

		// Заполнение таблиц из CSV
		err = populateDBFromCSVs(db, "dataset")
		if err != nil {
			db.Close()
			os.Remove(dbPath)
			return nil, fmt.Errorf("failed to populate database: %w", err)
		}
		fmt.Println("Database populated successfully.")
	} else {
		fmt.Println("Using existing database file.")
	}

	return db, nil
}

// populateDBFromCSVs управляет процессом загрузки данных из всех CSV-файлов.
func populateDBFromCSVs(db *sql.DB, dir string) error {
	loadOrder := []struct {
		File  string
		Table string
	}{
		{"circuits.csv", "circuits"},
		{"constructors.csv", "constructors"},
		{"drivers.csv", "drivers"},
		{"seasons.csv", "seasons"},
		{"status.csv", "status"},
		{"races.csv", "races"},
		{"results.csv", "results"},
		// Остальные файлы можно добавить здесь, если они нужны
		// {"lap_times.csv", "lap_times"},
		// {"pit_stops.csv", "pit_stops"},
		// {"qualifying.csv", "qualifying"},
		// ...
	}

	for _, item := range loadOrder {
		path := filepath.Join(dir, item.File)
		fmt.Printf("Loading %s into %s table...\n", item.File, item.Table)
		if err := loadCSVToTable(db, path, item.Table); err != nil {
			return fmt.Errorf("error loading %s: %w", path, err)
		}
	}
	return nil
}

func loadCSVToTable(db *sql.DB, filePath, tableName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("file %s is empty or has no header", filePath)
		}
		return fmt.Errorf("failed to read header from %s: %w", filePath, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	// Откатываем транзакцию в случае любой паники или ошибки
	defer tx.Rollback()

	query := fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s)",
		tableName,
		strings.Join(header, "\",\""),
		"?"+strings.Repeat(",?", len(header)-1))

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for table %s. Query: %s. Error: %w", tableName, query, err)
	}
	defer stmt.Close()

	recordCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading record from %s on line %d: %w", filePath, recordCount+2, err)
		}

		args := make([]interface{}, len(record))
		for i, v := range record {
			if v == `\N` {
				args[i] = nil
			} else {
				args[i] = v
			}
		}

		_, err = stmt.Exec(args...)
		if err != nil {
			return fmt.Errorf("failed to insert record into %s (line %d: %v): %w", tableName, recordCount+2, record, err)
		}
		recordCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for table %s: %w", tableName, err)
	}

	fmt.Printf("Successfully inserted %d records into table '%s'.\n", recordCount, tableName)
	return nil
}

func loadDataForYear(year string, statusUpdater binding.String, isFirstLoad bool) {
	statusUpdater.Set("Loading season data...")
	if !isFirstLoad {
		resetUIDataForDataView()
	}

	if !dbDataLoaded {
		err := loadAllDataFromDBSync(db)
		if err != nil {
			errMsg := "Error loading dataset: " + err.Error()
			statusUpdater.Set(errMsg)
			if !isFirstLoad {
				racesTabItem.Content = container.NewCenter(widget.NewLabel(errMsg))
				racesTabItem.Content.Refresh()
			}
			return
		}
	}

	if year == "" {
		statusUpdater.Set("Please enter a season year.")
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel("Please enter a year in the search bar above."))
			racesTabItem.Content.Refresh()
		}
		return
	}

	yearInt, err := strconv.Atoi(year)
	currentYear := time.Now().Year()
	if err != nil || yearInt < 1950 || yearInt > currentYear+10 {
		errMsg := fmt.Sprintf("Invalid season year. Must be a number between 1950 and %d.", currentYear+10)
		statusUpdater.Set(errMsg)
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(errMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}

	var currentSeasonRaces []RaceUI
	for _, raceDB := range allRacesData {
		if raceDB.Year == yearInt {
			uiRace := RaceUI{
				RaceIDInternal: raceDB.ID,
				Round:          strconv.Itoa(raceDB.Round),
				RaceName:       raceDB.Name,
				Date:           raceDB.Date.Format("2006-01-02"),
			}

			if raceDB.URL.Valid {
				uiRace.URL = raceDB.URL.String
			}

			if circuitDB, ok := allCircuitsData[raceDB.CircuitID]; ok {
				uiRace.Circuit = Circuit{
					ID:       circuitDB.ID,
					Ref:      circuitDB.Ref,
					Name:     circuitDB.Name,
					Location: circuitDB.Location,
					Country:  circuitDB.Country,
					Lat:      circuitDB.Lat,
					Lng:      circuitDB.Lng,
					Alt:      circuitDB.Alt,
					URL:      circuitDB.URL,
				}
			} else {
				uiRace.Circuit = Circuit{Name: "Unknown Circuit"}
				fmt.Printf("Warning: Circuit ID %s not found for race %s (ID: %s)\n", raceDB.CircuitID, raceDB.Name, raceDB.ID)
			}

			if raceDB.Fp1Date.Valid {
				uiRace.FirstPractice = Session{
					Date: raceDB.Fp1Date.Time.Format("2006-01-02"),
					Time: raceDB.Fp1Time.String,
				}
			}
			if raceDB.Fp2Date.Valid {
				uiRace.SecondPractice = Session{
					Date: raceDB.Fp2Date.Time.Format("2006-01-02"),
					Time: raceDB.Fp2Time.String,
				}
			}
			if raceDB.Fp3Date.Valid {
				uiRace.ThirdPractice = &Session{
					Date: raceDB.Fp3Date.Time.Format("2006-01-02"),
					Time: raceDB.Fp3Time.String,
				}
			}
			if raceDB.QualiDate.Valid {
				uiRace.Qualifying = Session{
					Date: raceDB.QualiDate.Time.Format("2006-01-02"),
					Time: raceDB.QualiTime.String,
				}
			}
			if raceDB.SprintDate.Valid {
				uiRace.Sprint = &Session{
					Date: raceDB.SprintDate.Time.Format("2006-01-02"),
					Time: raceDB.SprintTime.String,
				}
			}
			currentSeasonRaces = append(currentSeasonRaces, uiRace)
		}
	}

	sort.SliceStable(currentSeasonRaces, func(i, j int) bool {
		roundI, _ := strconv.Atoi(currentSeasonRaces[i].Round)
		roundJ, _ := strconv.Atoi(currentSeasonRaces[j].Round)
		return roundI < roundJ
	})

	if len(currentSeasonRaces) == 0 {
		errMsg := fmt.Sprintf("No races found for season %s in the dataset.", year)
		statusUpdater.Set(errMsg)
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(errMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}

	seasonWikiLink := widget.NewHyperlink("Season Info", nil)
	if seasonInfo, ok := allSeasonsData[year]; ok && seasonInfo.URL != "" {
		if parsedWikiURL := parseURL(seasonInfo.URL); parsedWikiURL != nil {
			seasonWikiLink.SetURL(parsedWikiURL)
		} else {
			seasonWikiLink.SetText("Season Info (URL error)")
		}
	} else {
		wikiURLStr := fmt.Sprintf("https://en.wikipedia.org/wiki/%s_Formula_One_World_Championship", year)
		if parsedWikiURL := parseURL(wikiURLStr); parsedWikiURL != nil {
			seasonWikiLink.SetURL(parsedWikiURL)
		} else {
			seasonWikiLink.SetText("Season Info (URL error)")
		}
	}

	races := currentSeasonRaces
	raceList := widget.NewList(
		func() int { return len(races) },
		func() fyne.CanvasObject { return widget.NewLabel("Race Template") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("Round %s: %s", races[id].Round, races[id].RaceName))
		},
	)
	raceInfoText := widget.NewLabel("Select a race to see details.")
	raceInfoText.Wrapping = fyne.TextWrapWord
	raceWikiLink := widget.NewHyperlink("", nil)
	raceDetailsContainer := container.NewVScroll(container.NewVBox(raceInfoText, raceWikiLink))
	split := container.NewHSplit(container.NewVScroll(raceList), raceDetailsContainer)
	split.SetOffset(0.3)
	topContentForRacesTab := container.NewVBox(seasonWikiLink, widget.NewSeparator())
	racesTabItem.Content = container.NewBorder(topContentForRacesTab, nil, nil, nil, split)
	racesTabItem.Content.Refresh()

	raceList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(races) {
			selectedRace := races[id]
			showRaceDetails(selectedRace, raceInfoText, raceWikiLink)
			loadRaceResults(selectedRace, resultsTable)
		}
	}

	if len(races) > 0 {
		raceList.Select(0)
	}

	loadDrivers(year, driversTable)
	loadConstructors(year, constructorsTable)

	statusUpdater.Set(fmt.Sprintf("Season %s loaded successfully!", year))
	if isFirstLoad {
		seasonEntryForDataView.SetText(year)
		statusTextForInputScreen.Set("")
		window.SetContent(dataViewScreen)
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		if window.Content() == dataViewScreen && tabs != nil {
			tabs.Refresh()
			resizeAllVisibleTables()
		}
	}()
}

func loadRaceResults(selectedRace RaceUI, table *widget.Table) {
	resetTable(table)
	numColsResults = 0
	if selectedRace.RaceIDInternal == "" || !dbDataLoaded {
		return
	}

	var raceResultsForDisplay []RaceResult
	for _, resultDB := range allResultsData {
		if resultDB.RaceID == selectedRace.RaceIDInternal {
			uiResult := RaceResult{
				Grid:   strconv.Itoa(resultDB.Grid),
				Laps:   strconv.Itoa(resultDB.Laps),
				Points: fmt.Sprintf("%.1f", resultDB.Points),
			}

			if resultDB.Number.Valid {
				uiResult.Number = strconv.Itoa(int(resultDB.Number.Int32))
			} else {
				uiResult.Number = ""
			}

			if resultDB.Position.Valid {
				uiResult.Position = strconv.Itoa(int(resultDB.Position.Int32))
			} else {
				uiResult.Position = resultDB.PositionText
			}

			if driverDB, ok := allDriversData[resultDB.DriverID]; ok {
				uiResult.Driver = Driver{
					ID:          driverDB.ID,
					Ref:         driverDB.Ref,
					Number:      driverDB.Number,
					Code:        driverDB.Code,
					Forename:    driverDB.Forename,
					Surname:     driverDB.Surname,
					DOB:         driverDB.DOB,
					Nationality: driverDB.Nationality,
					URL:         driverDB.URL,
				}
			} else {
				uiResult.Driver = Driver{Forename: "Unknown", Surname: "Driver"}
				fmt.Printf("Warning: Driver ID %s not found for result %s\n", resultDB.DriverID, resultDB.ID)
			}

			if constructorDB, ok := allConstructorsData[resultDB.ConstructorID]; ok {
				uiResult.Constructor = Constructor{
					ID:          constructorDB.ID,
					Ref:         constructorDB.Ref,
					Name:        constructorDB.Name,
					Nationality: constructorDB.Nationality,
					URL:         constructorDB.URL,
				}
			} else {
				uiResult.Constructor = Constructor{Name: "Unknown Team"}
				fmt.Printf("Warning: Constructor ID %s not found for result %s\n", resultDB.ConstructorID, resultDB.ID)
			}

			if statusDB, ok := allStatusesData[resultDB.StatusID]; ok {
				uiResult.Status = statusDB.Status
			} else {
				uiResult.Status = "Unknown"
				fmt.Printf("Warning: Status ID %s not found for result %s\n", resultDB.StatusID, resultDB.ID)
			}

			if resultDB.Time.Valid || resultDB.Milliseconds.Valid {
				uiResult.Time = &struct{ Millis, Time string }{}
				if resultDB.Time.Valid {
					uiResult.Time.Time = resultDB.Time.String
				}
				if resultDB.Milliseconds.Valid {
					uiResult.Time.Millis = strconv.Itoa(int(resultDB.Milliseconds.Int32))
				}
			}
			raceResultsForDisplay = append(raceResultsForDisplay, uiResult)
		}
	}

	if len(raceResultsForDisplay) == 0 {
		resetTable(table)
		return
	}

	results := raceResultsForDisplay
	headers := []string{"Pos", "No", "Driver", "Team", "Laps", "Time/Retired", "Status", "Points"}
	numColsResults = len(headers)
	table.Length = func() (int, int) { return len(results) + 1, numColsResults }
	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		label.Alignment = fyne.TextAlignLeading
		if id.Row == 0 {
			label.Alignment = fyne.TextAlignCenter
			if id.Col < len(headers) {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
			return
		}
		label.TextStyle = fyne.TextStyle{}
		resultIndex := id.Row - 1
		if resultIndex >= len(results) {
			return
		}
		result := results[resultIndex]
		switch id.Col {
		case 0:
			label.SetText(result.Position)
		case 1:
			label.SetText(result.Number)
		case 2:
			label.SetText(fmt.Sprintf("%s %s", result.Driver.Forename, result.Driver.Surname))
		case 3:
			label.SetText(result.Constructor.Name)
		case 4:
			label.SetText(result.Laps)
		case 5:
			if result.Time != nil && result.Time.Time != "" {
				label.SetText(result.Time.Time)
			} else {
				label.SetText(result.Status)
			}
		case 6:
			label.SetText(result.Status)
		case 7:
			label.SetText(result.Points)
		}
	}
	table.Refresh()
}

func loadDrivers(year string, table *widget.Table) {
	resetTable(table)
	numColsDrivers = 0
	if year == "" || !dbDataLoaded {
		return
	}

	driverIDsInSeason := make(map[string]bool)
	yearInt, _ := strconv.Atoi(year)
	for _, raceDB := range allRacesData {
		if raceDB.Year == yearInt {
			for _, resultDB := range allResultsData {
				if resultDB.RaceID == raceDB.ID {
					driverIDsInSeason[resultDB.DriverID] = true
				}
			}
		}
	}

	var driversForDisplay []Driver
	for driverID := range driverIDsInSeason {
		if driverDB, ok := allDriversData[driverID]; ok {
			driversForDisplay = append(driversForDisplay, driverDB)
		}
	}

	sort.Slice(driversForDisplay, func(i, j int) bool {
		if driversForDisplay[i].Surname != driversForDisplay[j].Surname {
			return driversForDisplay[i].Surname < driversForDisplay[j].Surname
		}
		return driversForDisplay[i].Forename < driversForDisplay[j].Forename
	})

	if len(driversForDisplay) == 0 {
		resetTable(table)
		return
	}

	drivers := driversForDisplay
	headers := []string{"Name", "Code", "Number", "Nationality", "DOB"}
	numColsDrivers = len(headers)
	table.Length = func() (int, int) { return len(drivers) + 1, numColsDrivers }
	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		label.Alignment = fyne.TextAlignLeading
		if id.Row == 0 {
			label.Alignment = fyne.TextAlignCenter
			if id.Col < len(headers) {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
			return
		}
		label.TextStyle = fyne.TextStyle{}
		driverIndex := id.Row - 1
		if driverIndex >= len(drivers) {
			return
		}
		driver := drivers[driverIndex]
		switch id.Col {
		case 0:
			label.SetText(fmt.Sprintf("%s %s", driver.Forename, driver.Surname))
		case 1:
			if driver.Code.Valid {
				label.SetText(driver.Code.String)
			} else {
				label.SetText("N/A")
			}
		case 2:
			if driver.Number.Valid {
				label.SetText(strconv.Itoa(int(driver.Number.Int32)))
			} else {
				label.SetText("N/A")
			}
		case 3:
			label.SetText(driver.Nationality)
		case 4:
			label.SetText(driver.DOB.Format("2006-01-02"))
		}
	}
	table.Refresh()
}

func loadConstructors(year string, table *widget.Table) {
	resetTable(table)
	numColsConstructors = 0
	if year == "" || !dbDataLoaded {
		return
	}

	constructorIDsInSeason := make(map[string]bool)
	yearInt, _ := strconv.Atoi(year)
	for _, raceDB := range allRacesData {
		if raceDB.Year == yearInt {
			for _, resultDB := range allResultsData {
				if resultDB.RaceID == raceDB.ID {
					constructorIDsInSeason[resultDB.ConstructorID] = true
				}
			}
		}
	}

	var constructorsForDisplay []Constructor
	for constructorID := range constructorIDsInSeason {
		if constructorDB, ok := allConstructorsData[constructorID]; ok {
			constructorsForDisplay = append(constructorsForDisplay, Constructor{
				ID:          constructorDB.ID,
				Ref:         constructorDB.Ref,
				Name:        constructorDB.Name,
				Nationality: constructorDB.Nationality,
				URL:         constructorDB.URL,
			})
		}
	}

	sort.Slice(constructorsForDisplay, func(i, j int) bool {
		return constructorsForDisplay[i].Name < constructorsForDisplay[j].Name
	})

	if len(constructorsForDisplay) == 0 {
		resetTable(table)
		return
	}

	constructors := constructorsForDisplay
	headers := []string{"Name", "Nationality"}
	numColsConstructors = len(headers)
	table.Length = func() (int, int) { return len(constructors) + 1, numColsConstructors }
	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		label.Alignment = fyne.TextAlignLeading
		if id.Row == 0 {
			label.Alignment = fyne.TextAlignCenter
			if id.Col < len(headers) {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
			return
		}
		label.TextStyle = fyne.TextStyle{}
		constructorIndex := id.Row - 1
		if constructorIndex >= len(constructors) {
			return
		}
		constructor := constructors[constructorIndex]
		switch id.Col {
		case 0:
			label.SetText(constructor.Name)
		case 1:
			label.SetText(constructor.Nationality)
		}
	}
	table.Refresh()
}

func resizeTableColumnsEqually(table *widget.Table, numCols int) {
	if table == nil || numCols == 0 || tabs == nil || !tabs.Visible() || window.Content() != dataViewScreen {
		return
	}
	if tabs.Size().Width <= 0 {
		return
	}
	availableWidth := tabs.Size().Width

	colWidth := availableWidth / float32(numCols)
	minPracticalWidth := float32(50)

	if colWidth < minPracticalWidth && float32(numCols)*minPracticalWidth > availableWidth {
		colWidth = minPracticalWidth
	} else if colWidth < 10 {
		colWidth = 10
	}

	for i := 0; i < numCols; i++ {
		table.SetColumnWidth(i, colWidth)
	}
	table.Refresh()
}

func resizeAllVisibleTables() {
	if window.Content() == dataViewScreen && tabs != nil && tabs.Selected() != nil {
		selectedTab := tabs.Selected()
		if len(tabs.Items) > 3 {
			if selectedTab == tabs.Items[1] {
				if resultsTable != nil && numColsResults > 0 {
					resizeTableColumnsEqually(resultsTable, numColsResults)
				}
			} else if selectedTab == tabs.Items[2] {
				if driversTable != nil && numColsDrivers > 0 {
					resizeTableColumnsEqually(driversTable, numColsDrivers)
				}
			} else if selectedTab == tabs.Items[3] {
				if constructorsTable != nil && numColsConstructors > 0 {
					resizeTableColumnsEqually(constructorsTable, numColsConstructors)
				}
			}
		}
	}
}

func resetUIDataForDataView() {
	if racesTabItem != nil {
		racesTabItem.Content = container.NewCenter(widget.NewLabel("Loading data or waiting for year input..."))
		racesTabItem.Content.Refresh()
	}
	if resultsTable != nil {
		resetTable(resultsTable)
	}
	if driversTable != nil {
		resetTable(driversTable)
	}
	if constructorsTable != nil {
		resetTable(constructorsTable)
	}
	numColsResults = 0
	numColsDrivers = 0
	numColsConstructors = 0
}

func resetTable(table *widget.Table) {
	table.Length = func() (int, int) { return 0, 0 }
	table.CreateCell = func() fyne.CanvasObject { return widget.NewLabel("") }
	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		if label, ok := cell.(*widget.Label); ok {
			label.SetText("")
		}
	}
	table.Refresh()
}

func showRaceDetails(race RaceUI, infoText *widget.Label, wikiLink *widget.Hyperlink) {
	text := fmt.Sprintf(
		"=== %s ===\nDate: %s\nCircuit: %s\nLocation: %s, %s\n",
		race.RaceName, race.Date, race.Circuit.Name,
		race.Circuit.Location,
		race.Circuit.Country,
	)

	if race.FirstPractice.Date != "" || race.FirstPractice.Time != "" {
		text += fmt.Sprintf("First Practice: %s %s\n", race.FirstPractice.Date, race.FirstPractice.Time)
	}
	if race.SecondPractice.Date != "" || race.SecondPractice.Time != "" {
		text += fmt.Sprintf("Second Practice: %s %s\n", race.SecondPractice.Date, race.SecondPractice.Time)
	}
	if race.ThirdPractice != nil && (race.ThirdPractice.Date != "" || race.ThirdPractice.Time != "") {
		text += fmt.Sprintf("Third Practice: %s %s\n", race.ThirdPractice.Date, race.ThirdPractice.Time)
	}
	if race.Qualifying.Date != "" || race.Qualifying.Time != "" {
		text += fmt.Sprintf("Qualifying: %s %s\n", race.Qualifying.Date, race.Qualifying.Time)
	}
	if race.Sprint != nil && (race.Sprint.Date != "" || race.Sprint.Time != "") {
		text += fmt.Sprintf("Sprint Race: %s %s", race.Sprint.Date, race.Sprint.Time)
	}

	if len(text) > 0 && text[len(text)-1] == '\n' {
		text = text[:len(text)-1]
	}
	infoText.SetText(text)

	if race.URL != "" {
		if parsedURL := parseURL(race.URL); parsedURL != nil {
			wikiLink.SetText("Wikipedia: " + race.RaceName)
			wikiLink.SetURL(parsedURL)
		} else {
			wikiLink.SetText("Race Info (URL error)")
			wikiLink.SetURL(nil)
		}
	} else {
		wikiLink.SetText("Race Info (URL not available)")
		wikiLink.SetURL(nil)
	}
	wikiLink.Refresh()
}

func parseURL(rawURL string) *url.URL {
	if rawURL == "" {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Error parsing URL:", rawURL, err)
		return nil
	}
	return u
}