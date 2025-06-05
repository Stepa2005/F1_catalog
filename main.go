package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Структуры для UI, они остаются, т.к. используются для отображения
type Circuit struct {
	CircuitID   string
	CircuitName string
	URL         string
	Location    Location
}

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

type Driver struct {
	DriverID        string
	PermanentNumber string
	Code            string
	URL             string
	GivenName       string
	FamilyName      string
	DateOfBirth     string
	Nationality     string
}

type Constructor struct {
	ConstructorID string
	URL           string
	Name          string
	Nationality   string
}

type Race struct {
	RaceIDInternal string // Внутренний ID гонки из CSV для связи
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

type RaceResult struct {
	Number      string
	Position    string
	Points      string
	Driver      Driver
	Constructor Constructor
	Grid        string
	Laps        string
	Status      string
	Time        *struct {
		Millis string
		Time   string
	}
}

// Глобальные переменные UI
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

func main() {
	myApp := app.New()

	err := loadAllDataFromCSVSync()
	if err != nil {
		fmt.Printf("CRITICAL: Failed to load initial CSV data: %v\n", err)
	}

	window = myApp.NewWindow("F1 Race Catalog (CSV Data)")
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
	tabs.Append(racesTabItem)                                                                  // Индекс 0
	tabs.Append(container.NewTabItem("Race Results", container.NewVScroll(resultsTable)))      // Индекс 1
	tabs.Append(container.NewTabItem("Drivers", container.NewVScroll(driversTable)))           // Индекс 2
	tabs.Append(container.NewTabItem("Constructors", container.NewVScroll(constructorsTable))) // Индекс 3

	tabs.OnSelected = func(tabItem *container.TabItem) {
		go func() {
			time.Sleep(100 * time.Millisecond)
			// Вызываем напрямую, Fyne должен обработать это в главном потоке
			resizeAllVisibleTables()
		}()
	}

	dataViewScreen = container.NewBorder(topPanelForDataView, nil, nil, nil, tabs)

	window.SetContent(inputScreen)
	window.ShowAndRun()
}

func loadDataForYear(year string, statusUpdater binding.String, isFirstLoad bool) {
	statusUpdater.Set("Loading season data...")
	if !isFirstLoad {
		resetUIDataForDataView()
	}

	if !csvDataLoaded {
		err := loadAllDataFromCSVSync()
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

	var currentSeasonRaces []Race
	for _, raceCSV := range allRacesData {
		if raceCSV.Year == year {
			uiRace := Race{
				RaceIDInternal: raceCSV.RaceID,
				Round:          raceCSV.Round,
				RaceName:       raceCSV.Name,
				Date:           raceCSV.Date,
				URL:            csvPointerToOptionalString(raceCSV.URL),
			}

			if circuitCSV, ok := allCircuitsData[raceCSV.CircuitID]; ok {
				uiRace.Circuit = Circuit{
					CircuitID:   circuitCSV.CircuitID,
					CircuitName: circuitCSV.Name,
					URL:         circuitCSV.URL,
					Location: Location{
						Lat:      circuitCSV.Lat,
						Long:     circuitCSV.Lng,
						Locality: circuitCSV.Location,
						Country:  circuitCSV.Country,
					},
				}
			} else {
				uiRace.Circuit = Circuit{CircuitName: "Unknown Circuit", Location: Location{Locality: "N/A", Country: "N/A"}}
				fmt.Printf("Warning: Circuit ID %s not found for race %s (ID: %s)\n", raceCSV.CircuitID, raceCSV.Name, raceCSV.RaceID)
			}

			if raceCSV.Fp1Date != nil && raceCSV.Fp1Time != nil {
				uiRace.FirstPractice = Session{Date: *raceCSV.Fp1Date, Time: *raceCSV.Fp1Time}
			}
			if raceCSV.Fp2Date != nil && raceCSV.Fp2Time != nil {
				uiRace.SecondPractice = Session{Date: *raceCSV.Fp2Date, Time: *raceCSV.Fp2Time}
			}
			if raceCSV.Fp3Date != nil && raceCSV.Fp3Time != nil {
				uiRace.ThirdPractice = &Session{Date: *raceCSV.Fp3Date, Time: *raceCSV.Fp3Time}
			}
			if raceCSV.QualiDate != nil && raceCSV.QualiTime != nil {
				uiRace.Qualifying = Session{Date: *raceCSV.QualiDate, Time: *raceCSV.QualiTime}
			}
			if raceCSV.SprintDate != nil && raceCSV.SprintTime != nil {
				uiRace.Sprint = &Session{Date: *raceCSV.SprintDate, Time: *raceCSV.SprintTime}
			}
			currentSeasonRaces = append(currentSeasonRaces, uiRace)
		}
	}

	sort.SliceStable(currentSeasonRaces, func(i, j int) bool {
		roundI, errI := strconv.Atoi(currentSeasonRaces[i].Round)
		roundJ, errJ := strconv.Atoi(currentSeasonRaces[j].Round)
		if errI == nil && errJ == nil {
			return roundI < roundJ
		}
		return currentSeasonRaces[i].Round < currentSeasonRaces[j].Round
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
			// Вызываем напрямую, Fyne должен обработать это в главном потоке
			resizeAllVisibleTables()
		}
	}()
}

func loadRaceResults(selectedRace Race, table *widget.Table) {
	resetTable(table)
	numColsResults = 0
	if selectedRace.RaceIDInternal == "" || !csvDataLoaded {
		return
	}

	var raceResultsForDisplay []RaceResult
	for _, resultCSV := range allResultsData {
		if resultCSV.RaceID == selectedRace.RaceIDInternal {
			uiResult := RaceResult{
				Grid:   resultCSV.Grid,
				Laps:   resultCSV.Laps,
				Points: resultCSV.Points,
				Number: csvPointerToDefaultString(resultCSV.Number, ""),
			}

			if resultCSV.Position != nil {
				uiResult.Position = *resultCSV.Position
				if uiResult.Position == `\N` {
					uiResult.Position = csvStringValueToDefault(resultCSV.PositionText, "N/A")
				}
			} else {
				uiResult.Position = csvStringValueToDefault(resultCSV.PositionText, "N/A")
			}

			if driverCSV, ok := allDriversData[resultCSV.DriverID]; ok {
				uiResult.Driver = Driver{
					DriverID:        driverCSV.DriverID,
					GivenName:       driverCSV.Forename,
					FamilyName:      driverCSV.Surname,
					Nationality:     driverCSV.Nationality,
					DateOfBirth:     driverCSV.DOB,
					URL:             driverCSV.URL,
					PermanentNumber: csvPointerToDefaultString(driverCSV.Number, ""),
					Code:            csvPointerToDefaultString(driverCSV.Code, ""),
				}
			} else {
				uiResult.Driver = Driver{GivenName: "Unknown", FamilyName: "Driver"}
				fmt.Printf("Warning: Driver ID %s not found for result %s\n", resultCSV.DriverID, resultCSV.ResultID)
			}

			if constructorCSV, ok := allConstructorsData[resultCSV.ConstructorID]; ok {
				uiResult.Constructor = Constructor{
					ConstructorID: constructorCSV.ConstructorID,
					Name:          constructorCSV.Name,
					Nationality:   constructorCSV.Nationality,
					URL:           constructorCSV.URL,
				}
			} else {
				uiResult.Constructor = Constructor{Name: "Unknown Team"}
				fmt.Printf("Warning: Constructor ID %s not found for result %s\n", resultCSV.ConstructorID, resultCSV.ResultID)
			}

			if statusCSV, ok := allStatusesData[resultCSV.StatusID]; ok {
				uiResult.Status = statusCSV.Status
			} else {
				uiResult.Status = "Unknown"
				fmt.Printf("Warning: Status ID %s not found for result %s\n", resultCSV.StatusID, resultCSV.ResultID)
			}

			if resultCSV.Time != nil || resultCSV.Milliseconds != nil {
				uiResult.Time = &struct{ Millis, Time string }{}
				if resultCSV.Time != nil {
					uiResult.Time.Time = *resultCSV.Time
				}
				if resultCSV.Milliseconds != nil {
					uiResult.Time.Millis = *resultCSV.Milliseconds
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
			label.SetText(fmt.Sprintf("%s %s", result.Driver.GivenName, result.Driver.FamilyName))
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
	if year == "" || !csvDataLoaded {
		return
	}

	driverIDsInSeason := make(map[string]bool)
	for _, raceCSV := range allRacesData {
		if raceCSV.Year == year {
			for _, resultCSV := range allResultsData {
				if resultCSV.RaceID == raceCSV.RaceID {
					driverIDsInSeason[resultCSV.DriverID] = true
				}
			}
		}
	}

	var driversForDisplay []Driver
	for driverID := range driverIDsInSeason {
		if driverCSV, ok := allDriversData[driverID]; ok {
			driversForDisplay = append(driversForDisplay, Driver{
				DriverID:        driverCSV.DriverID,
				GivenName:       driverCSV.Forename,
				FamilyName:      driverCSV.Surname,
				Nationality:     driverCSV.Nationality,
				DateOfBirth:     driverCSV.DOB,
				URL:             driverCSV.URL,
				PermanentNumber: csvPointerToDefaultString(driverCSV.Number, "N/A"),
				Code:            csvPointerToDefaultString(driverCSV.Code, "N/A"),
			})
		}
	}

	sort.Slice(driversForDisplay, func(i, j int) bool {
		if driversForDisplay[i].FamilyName != driversForDisplay[j].FamilyName {
			return driversForDisplay[i].FamilyName < driversForDisplay[j].FamilyName
		}
		return driversForDisplay[i].GivenName < driversForDisplay[j].GivenName
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
			label.SetText(fmt.Sprintf("%s %s", driver.GivenName, driver.FamilyName))
		case 1:
			label.SetText(driver.Code)
		case 2:
			label.SetText(driver.PermanentNumber)
		case 3:
			label.SetText(driver.Nationality)
		case 4:
			label.SetText(driver.DateOfBirth)
		}
	}
	table.Refresh()
}

func loadConstructors(year string, table *widget.Table) {
	resetTable(table)
	numColsConstructors = 0
	if year == "" || !csvDataLoaded {
		return
	}

	constructorIDsInSeason := make(map[string]bool)
	for _, raceCSV := range allRacesData {
		if raceCSV.Year == year {
			for _, resultCSV := range allResultsData {
				if resultCSV.RaceID == raceCSV.RaceID {
					constructorIDsInSeason[resultCSV.ConstructorID] = true
				}
			}
		}
	}

	var constructorsForDisplay []Constructor
	for constructorID := range constructorIDsInSeason {
		if constructorCSV, ok := allConstructorsData[constructorID]; ok {
			constructorsForDisplay = append(constructorsForDisplay, Constructor{
				ConstructorID: constructorCSV.ConstructorID,
				Name:          constructorCSV.Name,
				Nationality:   constructorCSV.Nationality,
				URL:           constructorCSV.URL,
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
		if len(tabs.Items) > 3 { // Убедимся что есть вкладки с индексами 1, 2, 3
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

func showRaceDetails(race Race, infoText *widget.Label, wikiLink *widget.Hyperlink) {
	text := fmt.Sprintf(
		"=== %s ===\nDate: %s\nCircuit: %s\nLocation: %s, %s\n",
		race.RaceName, race.Date, race.Circuit.CircuitName,
		csvStringValueToDefault(race.Circuit.Location.Locality, "N/A"),
		csvStringValueToDefault(race.Circuit.Location.Country, "N/A"),
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