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

// Структуры данных
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
	Time        *struct {
		Millis string `json:"millis"`
		Time   string `json:"time"`
	} `json:"Time,omitempty"`
}

func main() {
	myApp := app.New()
	window := myApp.NewWindow("F1 Race Catalog")
	window.Resize(fyne.NewSize(1200, 800))

	// UI элементы
	seasonEntry := widget.NewEntry()
	seasonEntry.SetPlaceHolder("Enter season year (e.g., 2023)")

	statusText := binding.NewString()
	statusLabel := widget.NewLabelWithData(statusText)
	statusLabel.Wrapping = fyne.TextWrapWord

	// Создаем таблицы
	resultsTable := widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)
	driversTable := widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)
	constructorsTable := widget.NewTable(
		func() (int, int) { return 0, 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {},
	)

	// Вкладки
	tabs := container.NewAppTabs()

	// Вкладка: Информация о сезоне
	wikiLink := widget.NewHyperlink("", nil)
	seasonInfo := widget.NewLabel("")
	seasonInfo.Wrapping = fyne.TextWrapWord

	// Создаем кнопку с иконкой лупы
	loadButton := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		loadSeasonData(
			seasonEntry.Text,
			statusText,
			wikiLink,
			seasonInfo,
			tabs,
			driversTable,
			constructorsTable,
			resultsTable,
		)
	})

	// Обработка нажатия Enter в поле ввода
	seasonEntry.OnSubmitted = func(_ string) {
		loadButton.OnTapped()
	}

	// Контейнер для поиска
	searchContainer := container.NewBorder(
		nil, nil,
		nil,
		loadButton,
		seasonEntry,
	)

	seasonTab := container.NewVBox(
		searchContainer,
		statusLabel,
		wikiLink,
		seasonInfo,
	)
	tabs.Append(container.NewTabItem("Season Info", seasonTab))

	// Вкладка: Календарь гонок - изначально пустая
	emptyRacesContainer := container.NewVBox()
	racesTab := container.NewTabItem("Races Calendar", emptyRacesContainer)
	tabs.Append(racesTab)

	// Добавляем вкладки с таблицами
	tabs.Append(container.NewTabItem("Race Results", container.NewVScroll(resultsTable)))
	tabs.Append(container.NewTabItem("Drivers", container.NewVScroll(driversTable)))
	tabs.Append(container.NewTabItem("Constructors", container.NewVScroll(constructorsTable)))

	window.SetContent(tabs)
	window.ShowAndRun()
}

// Функции загрузки данных
func loadSeasonData(
	year string,
	status binding.String,
	wikiLink *widget.Hyperlink,
	info *widget.Label,
	tabs *container.AppTabs,
	driversTable *widget.Table,
	constructorsTable *widget.Table,
	resultsTable *widget.Table,
) {
	// Сброс всех данных перед загрузкой новых
	resetAllData(wikiLink, info, tabs, driversTable, constructorsTable, resultsTable)

	if year == "" {
		status.Set("Please enter a season year")
		return
	}

	// Проверка корректности года
	yearInt, err := strconv.Atoi(year)
	currentYear := time.Now().Year()
	maxYear := currentYear - 1 // Установка верхней границы на предыдущий год
	if err != nil || yearInt < 1950 || yearInt > maxYear {
		status.Set(fmt.Sprintf("Invalid season year. Must be a number between 1950 and %d", maxYear))
		return
	}

	status.Set("Loading season data...")
	url := fmt.Sprintf("https://ergast.com/api/f1/%s.json", year)
	resp, err := http.Get(url)
	if err != nil {
		status.Set("Error fetching data: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		status.Set(fmt.Sprintf("Server error: %d", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		status.Set("Error reading response: " + err.Error())
		return
	}

	var data struct {
		MRData MRData `json:"MRData"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		status.Set("Error parsing JSON: " + err.Error())
		return
	}

	if len(data.MRData.RaceTable.Races) == 0 {
		status.Set("No races found for this season")
		return
	}

	// Обновление UI
	status.Set(fmt.Sprintf("Season %s loaded successfully!", year))
	wikiLink.SetText("Wikipedia: " + year + " Season")

	// Формируем корректную ссылку на Wikipedia
	wikiURL := fmt.Sprintf("https://en.wikipedia.org/wiki/%s_Formula_One_World_Championship", year)
	wikiLink.SetURL(parseURL(wikiURL))

	info.SetText(fmt.Sprintf(
		"Season %s: %d races\n"+
			"Series: %s",
		year,
		len(data.MRData.RaceTable.Races),
		data.MRData.Series,
	))

	// Создаем содержимое для вкладки календаря
	raceList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {},
	)

	raceInfoText := widget.NewLabel("")
	raceInfoText.Wrapping = fyne.TextWrapWord
	raceWikiLink := widget.NewHyperlink("", nil)

	raceDetailsContainer := container.NewVBox(
		raceInfoText,
		raceWikiLink,
	)

	split := container.NewHSplit(
		container.NewVScroll(raceList),
		container.NewVScroll(raceDetailsContainer),
	)
	split.SetOffset(0.3)

	// Обновляем вкладку календаря
	racesTabIndex := 1
	tabs.Items[racesTabIndex].Content = split

	// Обновление списка гонок
	raceTitles := make([]string, len(data.MRData.RaceTable.Races))
	for i, race := range data.MRData.RaceTable.Races {
		raceTitles[i] = fmt.Sprintf("Round %s: %s", race.Round, race.RaceName)
	}

	raceList.Length = func() int { return len(raceTitles) }
	raceList.UpdateItem = func(id widget.ListItemID, item fyne.CanvasObject) {
		item.(*widget.Label).SetText(raceTitles[id])
	}

	// Обработчик выбора гонки
	raceList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(data.MRData.RaceTable.Races) {
			showRaceDetails(data.MRData.RaceTable.Races[id], raceInfoText, raceWikiLink)
			loadRaceResults(year, data.MRData.RaceTable.Races[id].Round, resultsTable)
		}
	}

	// Выбор первой гонки
	if len(data.MRData.RaceTable.Races) > 0 {
		raceList.Select(0)
	}

	// Загрузка дополнительных данных
	loadDrivers(year, driversTable)
	loadConstructors(year, constructorsTable)
}

// Сброс всех данных при загрузке нового сезона
func resetAllData(
	wikiLink *widget.Hyperlink,
	info *widget.Label,
	tabs *container.AppTabs,
	driversTable *widget.Table,
	constructorsTable *widget.Table,
	resultsTable *widget.Table,
) {
	// Сбрасываем данные на вкладке сезона
	wikiLink.SetText("")
	wikiLink.SetURL(nil)
	info.SetText("")

	// Сбрасываем вкладку календаря гонок
	racesTabIndex := 1
	tabs.Items[racesTabIndex].Content = container.NewVBox()

	// Сбрасываем таблицы
	resetTable(driversTable)
	resetTable(constructorsTable)
	resetTable(resultsTable)
}

// Сброс таблицы
func resetTable(table *widget.Table) {
	table.Length = func() (int, int) { return 0, 0 }
	table.Refresh()
}

func showRaceDetails(race Race, infoText *widget.Label, wikiLink *widget.Hyperlink) {
	text := fmt.Sprintf(
		"=== %s ===\n"+
			"Date: %s\n"+
			"Circuit: %s\n"+
			"Location: %s, %s\n"+
			"First Practice: %s %s\n"+
			"Second Practice: %s %s\n",
		race.RaceName,
		race.Date,
		race.Circuit.CircuitName,
		race.Circuit.Location.Locality,
		race.Circuit.Location.Country,
		race.FirstPractice.Date, race.FirstPractice.Time,
		race.SecondPractice.Date, race.SecondPractice.Time,
	)

	if race.ThirdPractice != nil {
		text += fmt.Sprintf("Third Practice: %s %s\n",
			race.ThirdPractice.Date, race.ThirdPractice.Time)
	}

	text += fmt.Sprintf("Qualifying: %s %s\n",
		race.Qualifying.Date, race.Qualifying.Time)

	if race.Sprint != nil {
		text += fmt.Sprintf("Sprint Race: %s %s",
			race.Sprint.Date, race.Sprint.Time)
	}

	// Убираем последний перевод строки если он есть
	if len(text) > 0 && text[len(text)-1] == '\n' {
		text = text[:len(text)-1]
	}

	infoText.SetText(text)
	wikiLink.SetText("Wikipedia: " + race.RaceName)
	wikiLink.SetURL(parseURL(race.URL))
}

func loadRaceResults(year, round string, table *widget.Table) {
	if year == "" || round == "" {
		return
	}

	url := fmt.Sprintf("https://ergast.com/api/f1/%s/%s/results.json", year, round)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
		return
	}

	if len(resultData.MRData.RaceTable.Races) == 0 || len(resultData.MRData.RaceTable.Races[0].Results) == 0 {
		resetTable(table)
		return
	}

	results := resultData.MRData.RaceTable.Races[0].Results
	headers := []string{"Pos", "No", "Driver", "Team", "Laps", "Time", "Status", "Points"}

	table.Length = func() (int, int) {
		return len(results) + 1, len(headers)
	}

	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		label.Alignment = fyne.TextAlignCenter

		if id.Row == 0 {
			if id.Col < len(headers) {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
			return
		}

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

	for i := 0; i < len(headers); i++ {
		table.SetColumnWidth(i, 120)
	}

	table.Refresh()
}

func loadDrivers(year string, table *widget.Table) {
	if year == "" {
		resetTable(table)
		return
	}

	url := fmt.Sprintf("https://ergast.com/api/f1/%s/drivers.json", year)
	resp, err := http.Get(url)
	if err != nil {
		resetTable(table)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		resetTable(table)
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
		resetTable(table)
		return
	}

	if len(driverData.MRData.DriverTable.Drivers) == 0 {
		resetTable(table)
		return
	}

	headers := []string{"Name", "Code", "Number", "Nationality", "DOB"}

	table.Length = func() (int, int) {
		return len(driverData.MRData.DriverTable.Drivers) + 1, len(headers)
	}

	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		label.Alignment = fyne.TextAlignCenter

		if id.Row == 0 {
			if id.Col < len(headers) {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
			return
		}

		driverIndex := id.Row - 1
		if driverIndex >= len(driverData.MRData.DriverTable.Drivers) {
			return
		}

		driver := driverData.MRData.DriverTable.Drivers[driverIndex]
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

	table.SetColumnWidth(0, 180)
	table.SetColumnWidth(1, 80)
	table.SetColumnWidth(2, 80)
	table.SetColumnWidth(3, 120)
	table.SetColumnWidth(4, 120)

	table.Refresh()
}

func loadConstructors(year string, table *widget.Table) {
	if year == "" {
		resetTable(table)
		return
	}

	url := fmt.Sprintf("https://ergast.com/api/f1/%s/constructors.json", year)
	resp, err := http.Get(url)
	if err != nil {
		resetTable(table)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		resetTable(table)
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
		resetTable(table)
		return
	}

	if len(constructorData.MRData.ConstructorTable.Constructors) == 0 {
		resetTable(table)
		return
	}

	headers := []string{"Name", "Nationality"}

	table.Length = func() (int, int) {
		return len(constructorData.MRData.ConstructorTable.Constructors) + 1, len(headers)
	}

	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		label.Alignment = fyne.TextAlignCenter

		if id.Row == 0 {
			if id.Col < len(headers) {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
			}
			return
		}

		constructorIndex := id.Row - 1
		if constructorIndex >= len(constructorData.MRData.ConstructorTable.Constructors) {
			return
		}

		constructor := constructorData.MRData.ConstructorTable.Constructors[constructorIndex]
		switch id.Col {
		case 0:
			label.SetText(constructor.Name)
		case 1:
			label.SetText(constructor.Nationality)
		}
	}

	table.SetColumnWidth(0, 200)
	table.SetColumnWidth(1, 150)

	table.Refresh()
}

// Вспомогательные функции
func parseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	return u
}
