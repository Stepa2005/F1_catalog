-- Создаем схему для данных F1
CREATE SCHEMA f1;

-- Таблица Circuits
CREATE TABLE f1.circuits (
    circuit_id TEXT PRIMARY KEY,
    circuit_ref TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    country TEXT NOT NULL,
    lat TEXT,
    lng TEXT,
    alt TEXT,
    url TEXT NOT NULL
);

-- Таблица Constructors
CREATE TABLE f1.constructors (
    constructor_id TEXT PRIMARY KEY,
    constructor_ref TEXT NOT NULL,
    name TEXT NOT NULL,
    nationality TEXT NOT NULL,
    url TEXT NOT NULL
);

-- Таблица Drivers
CREATE TABLE f1.drivers (
    driver_id TEXT PRIMARY KEY,
    driver_ref TEXT NOT NULL,
    number TEXT,
    code TEXT,
    forename TEXT NOT NULL,
    surname TEXT NOT NULL,
    dob DATE NOT NULL,
    nationality TEXT NOT NULL,
    url TEXT NOT NULL
);

-- Таблица Races
CREATE TABLE f1.races (
    race_id TEXT PRIMARY KEY,
    year INTEGER NOT NULL,
    round INTEGER NOT NULL,
    circuit_id TEXT REFERENCES f1.circuits(circuit_id),
    name TEXT NOT NULL,
    date DATE NOT NULL,
    time TIME,
    url TEXT,
    fp1_date DATE,
    fp1_time TIME,
    fp2_date DATE,
    fp2_time TIME,
    fp3_date DATE,
    fp3_time TIME,
    quali_date DATE,
    quali_time TIME,
    sprint_date DATE,
    sprint_time TIME
);

-- Таблица Results
CREATE TABLE f1.results (
    result_id TEXT PRIMARY KEY,
    race_id TEXT REFERENCES f1.races(race_id),
    driver_id TEXT REFERENCES f1.drivers(driver_id),
    constructor_id TEXT REFERENCES f1.constructors(constructor_id),
    number INTEGER,
    grid INTEGER NOT NULL,
    position INTEGER,
    position_text TEXT NOT NULL,
    position_order INTEGER NOT NULL,
    points FLOAT NOT NULL,
    laps INTEGER NOT NULL,
    time TEXT,
    milliseconds INTEGER,
    fastest_lap INTEGER,
    rank INTEGER,
    fastest_lap_time TEXT,
    fastest_lap_speed FLOAT,
    status_id TEXT NOT NULL
);

-- Таблица Status
CREATE TABLE f1.status (
    status_id TEXT PRIMARY KEY,
    status TEXT NOT NULL
);

-- Таблица Seasons
CREATE TABLE f1.seasons (
    year INTEGER PRIMARY KEY,
    url TEXT NOT NULL
);