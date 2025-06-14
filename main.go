package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MRData struct {
	Xmlns     string    `json:"xmlns"`
	Series    string    `json:"series"`
	URL       string    `json:"url"`
	RaceTable RaceTable `json:"RaceTable"`
}

type RaceTable struct {
	Season string `json:"season"`
	Races  []Race `json:"Races"`
}

type Circuit struct {
	CircuitID   string   `json:"circuitId"`
	CircuitName string   `json:"circuitName"`
	URL         string   `json:"url"`
	Location    Location `json:"Location"`
}

type Location struct {
	Lat      string `json:"lat"`
	Long     string `json:"long"`
	Locality string `json:"locality"`
	Country  string `json:"country"`
}

type Session struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

type Driver struct {
	DriverID        string `json:"driverId"`
	PermanentNumber string `json:"permanentNumber"`
	Code            string `json:"code"`
	URL             string `json:"url"`
	GivenName       string `json:"givenName"`
	FamilyName      string `json:"familyName"`
	DateOfBirth     string `json:"dateOfBirth"`
	Nationality     string `json:"nationality"`
}

type Constructor struct {
	ConstructorID string `json:"constructorId"`
	URL           string `json:"url"`
	Name          string `json:"name"`
	Nationality   string `json:"nationality"`
}

type Race struct {
	Round          string   `json:"round"`
	RaceName       string   `json:"raceName"`
	Date           string   `json:"date"`
	Circuit        Circuit  `json:"Circuit"`
	URL            string   `json:"url"`
	FirstPractice  Session  `json:"FirstPractice"`
	SecondPractice Session  `json:"SecondPractice"`
	ThirdPractice  *Session `json:"ThirdPractice,omitempty"`
	Qualifying     Session  `json:"Qualifying"`
	Sprint         *Session `json:"Sprint,omitempty"`
}

type RaceResult struct {
	Number      string      `json:"number"`
	Position    string      `json:"position"`
	Points      string      `json:"points"`
	Driver      Driver      `json:"Driver"`
	Constructor Constructor `json:"Constructor"`
	Grid        string      `json:"grid"`
	Laps        string      `json:"laps"`
	Status      string      `json:"status"`
	Time        *struct { //анонимная структура
		Millis string `json:"millis"`
		Time   string `json:"time"`
	} `json:"Time,omitempty"` //если Time == nil, то при сериализации в JSON это поле будет пропущено
}

var ( // глобальные переменные
	window fyne.Window // главное окно приложения
	tabs   *container.AppTabs // вкладки "Races Calendar", "Race Results", "Drivers", "Constructors"

	racesTabItem *container.TabItem // вкладка, где отображается список гонок и их детали

	resultsTable      *widget.Table
	driversTable      *widget.Table
	constructorsTable *widget.Table

	numColsResults      int
	numColsDrivers      int
	numColsConstructors int

	seasonEntryForInputScreen *widget.Entry
	statusTextForInputScreen  binding.String // текст состояния (ввод/ошибка/загрузка)
	statusLabelForInputScreen *widget.Label

	seasonEntryForDataView *widget.Entry
	statusTextForDataView  binding.String
	statusLabelForDataView *widget.Label

	inputScreen    fyne.CanvasObject // первый экран
	dataViewScreen fyne.CanvasObject // основной экран
)

func main() {
	myApp := app.New()
	window = myApp.NewWindow("F1 Race Catalog")
	window.Resize(fyne.NewSize(1200, 700))

	// Инициализация элементов для экрана ввода
	seasonEntryForInputScreen = widget.NewEntry()
	seasonEntryForInputScreen.SetPlaceHolder("Enter season year (e.g., 2023)")

	statusTextForInputScreen = binding.NewString() // автоматически обновляет текст при изменении переменной
	statusLabelForInputScreen = widget.NewLabelWithData(statusTextForInputScreen)
	statusLabelForInputScreen.Wrapping = fyne.TextWrapWord

	loadButtonForInputScreen := widget.NewButtonWithIcon("Load Data", theme.SearchIcon(), func() {
		loadDataForYear(seasonEntryForInputScreen.Text, statusTextForInputScreen, true)
	})
	seasonEntryForInputScreen.OnSubmitted = func(_ string) { loadButtonForInputScreen.OnTapped() }

	// Создаем контейнер с фиксированной шириной поля ввода
	wrappedEntry := container.NewGridWrap(fyne.NewSize(300, seasonEntryForInputScreen.MinSize().Height), seasonEntryForInputScreen)
	inputContainer := container.NewVBox( // вертикально выравниваем
		widget.NewLabel("Enter Formula 1 Season Year"),
		wrappedEntry,
		loadButtonForInputScreen,
		statusLabelForInputScreen,
	)
	inputScreen = container.NewCenter(inputContainer) // централизуем

	// Инициализация элементов для основного экрана
	seasonEntryForDataView = widget.NewEntry()
	seasonEntryForDataView.SetPlaceHolder("Enter another year...")

	statusTextForDataView = binding.NewString()
	statusLabelForDataView = widget.NewLabelWithData(statusTextForDataView)
	statusLabelForDataView.Wrapping = fyne.TextWrapWord // Если строка не помещается по ширине — перености её по словам на новую строку

	loadButtonForDataView := widget.NewButtonWithIcon("Load New Season", theme.SearchIcon(), func() {
		loadDataForYear(seasonEntryForDataView.Text, statusTextForDataView, false)
	})
	seasonEntryForDataView.OnSubmitted = func(_ string) { loadButtonForDataView.OnTapped() }

	searchBarForDataView := container.NewBorder(nil, nil, nil, loadButtonForDataView, seasonEntryForDataView)
	topPanelForDataView := container.NewVBox(searchBarForDataView, statusLabelForDataView)

	// Инициализация таблиц
	resultsTable = widget.NewTable(
		func() (int, int) { return 0, 0 }, // строки, столбцы
		func() fyne.CanvasObject { return widget.NewLabel("") }, // что писать
		func(id widget.TableCellID, cell fyne.CanvasObject) {}, // вызывается каждый раз, когда нужно обновить содержимое конкретной ячейки
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

	// Создание вкладок
	tabs = container.NewAppTabs()
	initialRacesContent := container.NewCenter(widget.NewLabel("Data will appear here after loading a season."))
	racesTabItem = container.NewTabItem("Races Calendar", initialRacesContent)
	tabs.Append(racesTabItem)
	tabs.Append(container.NewTabItem("Race Results", container.NewVScroll(resultsTable)))
	tabs.Append(container.NewTabItem("Drivers", container.NewVScroll(driversTable)))
	tabs.Append(container.NewTabItem("Constructors", container.NewVScroll(constructorsTable)))

	dataViewScreen = container.NewBorder(topPanelForDataView, nil, nil, nil, tabs)

	window.SetContent(inputScreen)
	window.ShowAndRun()
}
func resizeTableColumnsEqually(table *widget.Table, numCols int) {
	if table == nil || numCols == 0 || tabs == nil || !tabs.Visible() || window.Content() != dataViewScreen {
		return
	}
	availableWidth := tabs.Size().Width
	if availableWidth <= 0 {
		// fmt.Println("resizeTableColumnsEqually: availableWidth for tabs is 0 or less, cannot resize.")
		return
	}

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
}

func loadDataForYear(year string, statusUpdater binding.String, isFirstLoad bool) {
	statusUpdater.Set("Loading season data...")
	if !isFirstLoad { // если не на стартовом экране - очищаем старые данные
		resetUIDataForDataView()
	}

	placeholderErrorMsg := "Error loading data. Please check the year or your connection."
	if year == "" {
		statusUpdater.Set("Please enter a season year.")
		if !isFirstLoad { // Обновляем вкладку с гонками если это не первый экран
			racesTabItem.Content = container.NewCenter(widget.NewLabel("Please enter a year in the search bar above."))
			racesTabItem.Content.Refresh()
		}
		return
	}

	yearInt, err := strconv.Atoi(year)
	currentYear := time.Now().Year()
	if err != nil || yearInt < 1950 || yearInt > currentYear {
		errMsg := fmt.Sprintf("Invalid season year. Must be a number between 1950 and %d.", currentYear)
		statusUpdater.Set(errMsg)
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(errMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}

	apiUrl := fmt.Sprintf("https://ergast.com/api/f1/%s.json", year)
	resp, err := http.Get(apiUrl)
	if err != nil {
		statusUpdater.Set("Error fetching data: " + err.Error())
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(placeholderErrorMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Server error: %d. No data or try a different year.", resp.StatusCode)
		statusUpdater.Set(errMsg)
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(errMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}

	body, err := io.ReadAll(resp.Body) // читаем ответ и раскладываем по структурам
	if err != nil {
		statusUpdater.Set("Error reading response: " + err.Error())
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(placeholderErrorMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}

	var apiResponseData struct {
		MRData MRData `json:"MRData"`
	}
	if err := json.Unmarshal(body, &apiResponseData); err != nil {
		statusUpdater.Set("Error parsing JSON: " + err.Error())
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(placeholderErrorMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}

	if len(apiResponseData.MRData.RaceTable.Races) == 0 { // если гонок нет, то сообщаем об этом
		errMsg := fmt.Sprintf("No races found for season %s.", year)
		statusUpdater.Set(errMsg)
		if !isFirstLoad {
			racesTabItem.Content = container.NewCenter(widget.NewLabel(errMsg))
			racesTabItem.Content.Refresh()
		}
		return
	}

	seasonWikiLink := widget.NewHyperlink("Season Info", nil)
	wikiURLStr := fmt.Sprintf("https://en.wikipedia.org/wiki/%s_Formula_One_World_Championship", year)
	parsedWikiURL := parseURL(wikiURLStr)
	if parsedWikiURL != nil {
		seasonWikiLink.SetURL(parsedWikiURL)
	} else {
		seasonWikiLink.SetText("Season Info (URL error)")
	}

	// Создаем список гонок, поле для информации о выбранной гонке и ссылку на Википедию
	races := apiResponseData.MRData.RaceTable.Races
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

	raceList.OnSelected = func(id widget.ListItemID) { // показываем гонку на которую нажал пользователь
		if id >= 0 && id < len(races) {
			selectedRace := races[id]
			showRaceDetails(selectedRace, raceInfoText, raceWikiLink)
			loadRaceResults(year, selectedRace.Round, resultsTable)
		}
	}
	if len(races) > 0 {
		raceList.Select(0)
	}

	// Загружаем списки пилотов и конструкторов на отдельные вкладки
	loadDrivers(year, driversTable)
	loadConstructors(year, constructorsTable)

	// Если это первый запуск — переключаемся с input-экрана на экран с вкладками
	statusUpdater.Set(fmt.Sprintf("Season %s loaded successfully!", year))
	if isFirstLoad {
		seasonEntryForDataView.SetText(year)
		statusTextForInputScreen.Set("")
		window.SetContent(dataViewScreen)
	}

	// Горутина тут нужна, чтобы не блокировался UI и выполнилось обновление интерфейса после задержки
	go func() { 	// Через полсекунды обновляем ширину колонок, чтобы красиво отображались данные
		time.Sleep(150 * time.Millisecond) // Даем Fyne время отрисовать и обновить размеры
		if window.Content() == dataViewScreen && tabs != nil {
			tabs.Refresh() // Обновляем сам контейнер вкладок, чтобы его Size() был актуален
		}
		resizeAllVisibleTables()
	}() // сразу ее вызываем
}

func resizeAllVisibleTables() {
	if window.Content() == dataViewScreen {
		if resultsTable != nil && numColsResults > 0 {
			resizeTableColumnsEqually(resultsTable, numColsResults)
		}
		if driversTable != nil && numColsDrivers > 0 {
			resizeTableColumnsEqually(driversTable, numColsDrivers)
		}
		if constructorsTable != nil && numColsConstructors > 0 {
			resizeTableColumnsEqually(constructorsTable, numColsConstructors)
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
	table.Refresh()
}

func showRaceDetails(race Race, infoText *widget.Label, wikiLink *widget.Hyperlink) {
	text := fmt.Sprintf(
		"=== %s ===\nDate: %s\nCircuit: %s\nLocation: %s, %s\nFirst Practice: %s %s\nSecond Practice: %s %s\n",
		race.RaceName, race.Date, race.Circuit.CircuitName, race.Circuit.Location.Locality, race.Circuit.Location.Country,
		race.FirstPractice.Date, race.FirstPractice.Time, race.SecondPractice.Date, race.SecondPractice.Time,
	)
	if race.ThirdPractice != nil {
		text += fmt.Sprintf("Third Practice: %s %s\n", race.ThirdPractice.Date, race.ThirdPractice.Time)
	}
	text += fmt.Sprintf("Qualifying: %s %s\n", race.Qualifying.Date, race.Qualifying.Time)
	if race.Sprint != nil {
		text += fmt.Sprintf("Sprint Race: %s %s", race.Sprint.Date, race.Sprint.Time)
	}
	if len(text) > 0 && text[len(text)-1] == '\n' {
		text = text[:len(text)-1]
	}
	infoText.SetText(text)
	wikiLink.SetText("Wikipedia: " + race.RaceName)
	wikiLink.SetURL(parseURL(race.URL))
}

func loadRaceResults(year, round string, table *widget.Table) {
	resetTable(table)
	numColsResults = 0
	if year == "" || round == "" {
		return
	}
	url := fmt.Sprintf("https://ergast.com/api/f1/%s/%s/results.json", year, round)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching race results:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Server error fetching race results:", resp.StatusCode)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	var resultData struct {
		MRData struct {
			RaceTable struct {
				Races []struct {
					Results []RaceResult `json:"Results"`
				} `json:"Races"`
			} `json:"RaceTable"`
		} `json:"MRData"`
	}
	if err := json.Unmarshal(body, &resultData); err != nil {
		fmt.Println("Error parsing race results JSON:", err)
		return
	}
	if len(resultData.MRData.RaceTable.Races) == 0 || len(resultData.MRData.RaceTable.Races[0].Results) == 0 {
		resetTable(table)  // Если результатов нет — таблица остается пустой
		return
	}
	results := resultData.MRData.RaceTable.Races[0].Results
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
		label.TextStyle = fyne.TextStyle{}  //Если Row == 0, это строка заголовков - пишем bold-текст, если Row > 0, это данные гонки
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
	table.Refresh() // перерисовываем таблицу на экране
}

func loadDrivers(year string, table *widget.Table) {
	resetTable(table)
	numColsDrivers = 0
	if year == "" {
		return
	}
	url := fmt.Sprintf("https://ergast.com/api/f1/%s/drivers.json", year)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching drivers:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Server error fetching drivers:", resp.StatusCode)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	var driverData struct {
		MRData struct {
			DriverTable struct {
				Drivers []Driver `json:"Drivers"`
			} `json:"DriverTable"`
		} `json:"MRData"`
	}
	if err := json.Unmarshal(body, &driverData); err != nil {
		fmt.Println("Error parsing drivers JSON:", err)
		return
	}
	if len(driverData.MRData.DriverTable.Drivers) == 0 {
		resetTable(table)
		return
	}
	drivers := driverData.MRData.DriverTable.Drivers
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
	if year == "" {
		return
	}
	url := fmt.Sprintf("https://ergast.com/api/f1/%s/constructors.json", year)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching constructors:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Server error fetching constructors:", resp.StatusCode)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	var constructorData struct {
		MRData struct {
			ConstructorTable struct {
				Constructors []Constructor `json:"Constructors"`
			} `json:"ConstructorTable"`
		} `json:"MRData"`
	}
	if err := json.Unmarshal(body, &constructorData); err != nil {
		fmt.Println("Error parsing constructors JSON:", err)
		return
	}
	if len(constructorData.MRData.ConstructorTable.Constructors) == 0 {
		resetTable(table)
		return
	}
	constructors := constructorData.MRData.ConstructorTable.Constructors
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

func parseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Error parsing URL:", rawURL, err)
		return nil
	}
	return u
}
