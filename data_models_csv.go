package main

// --- Глобальные хранилища для данных из CSV ---
var (
	allCircuitsData     map[string]CircuitCSVRecord
	allConstructorsData map[string]ConstructorCSVRecord
	allDriversData      map[string]DriverCSVRecord
	allRacesData        []RaceCSVRecord
	allResultsData      []ResultCSVRecord
	allStatusesData     map[string]StatusCSVRecord
	allSeasonsData      map[string]SeasonCSVRecord
	csvDataLoaded       bool = false
)

// --- Вспомогательные функции для парсинга и обработки значений из CSV ---

// csvPointerToOptionalString: для полей *string из CSV.
// Если указатель nil или указывает на "\N", возвращает пустую строку.
// Иначе возвращает разыменованное значение.
func csvPointerToOptionalString(s *string) string {
	if s == nil || *s == `\N` {
		return ""
	}
	return *s
}

// csvPointerToDefaultString: для полей *string из CSV.
// Если указатель nil или указывает на "\N", возвращает defaultValue.
// Иначе возвращает разыменованное значение.
func csvPointerToDefaultString(s *string, defaultValue string) string {
	if s == nil || *s == `\N` {
		return defaultValue
	}
	return *s
}

// csvStringValueToDefault: для полей string (например, уже обработанных или не допускающих \N).
// Если строка равна "\N" или пуста "", возвращает defaultValue.
// Иначе возвращает исходную строку.
func csvStringValueToDefault(s string, defaultValue string) string {
	if s == `\N` || s == "" {
		return defaultValue
	}
	return s
}

// csvOptionalString: Используется при парсинге CSV для создания *string.
// Если строка из CSV это `\N` или пустая, возвращает nil.
// Иначе возвращает указатель на строку.
func csvOptionalString(s string) *string {
	if s == `\N` || s == "" {
		return nil
	}
	return &s
}


// --- Структуры для данных из CSV файлов ---

type CircuitCSVRecord struct {
	CircuitID  string
	CircuitRef string
	Name       string
	Location   string // Locality
	Country    string
	Lat        string
	Lng        string
	Alt        *string // Optional
	URL        string
}

type ConstructorCSVRecord struct {
	ConstructorID  string
	ConstructorRef string
	Name           string
	Nationality    string
	URL            string
}

type DriverCSVRecord struct {
	DriverID    string
	DriverRef   string
	Number      *string // Может быть \N
	Code        *string // Может быть \N
	Forename    string
	Surname     string
	DOB         string
	Nationality string
	URL         string
}

type RaceCSVRecord struct {
	RaceID     string
	Year       string
	Round      string
	CircuitID  string
	Name       string // Race Name
	Date       string
	Time       *string // Может быть \N
	URL        *string // Может быть \N
	Fp1Date    *string
	Fp1Time    *string
	Fp2Date    *string
	Fp2Time    *string
	Fp3Date    *string
	Fp3Time    *string
	QualiDate  *string
	QualiTime  *string
	SprintDate *string
	SprintTime *string
}

type ResultCSVRecord struct {
	ResultID        string
	RaceID          string
	DriverID        string
	ConstructorID   string
	Number          *string // Может быть \N
	Grid            string
	Position        *string // Может быть \N для DNF, DSQ и т.д.
	PositionText    string  // Например, "1", "R"
	PositionOrder   string  // Для сортировки
	Points          string
	Laps            string
	Time            *string // Время гонки или отрыв, может быть \N
	Milliseconds    *string // Может быть \N
	FastestLap      *string // Номер круга, может быть \N
	Rank            *string // Ранг быстрейшего круга, может быть \N
	FastestLapTime  *string // Может быть \N
	FastestLapSpeed *string // Может быть \N
	StatusID        string
}

type StatusCSVRecord struct {
	StatusID string
	Status   string
}

type SeasonCSVRecord struct {
	Year string
	URL  string
}