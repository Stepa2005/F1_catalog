-- schema.sql (ИСПРАВЛЕННАЯ ВЕРСИЯ)

CREATE TABLE circuits (
    circuitId TEXT PRIMARY KEY,
    circuitRef TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    country TEXT NOT NULL,
    lat TEXT,
    lng TEXT,
    alt TEXT,
    url TEXT NOT NULL
);

CREATE TABLE constructors (
    constructorId TEXT PRIMARY KEY,
    constructorRef TEXT NOT NULL,
    name TEXT NOT NULL,
    nationality TEXT NOT NULL,
    url TEXT NOT NULL
);

CREATE TABLE drivers (
    driverId TEXT PRIMARY KEY,
    driverRef TEXT NOT NULL,
    number INTEGER,
    code TEXT,
    forename TEXT NOT NULL,
    surname TEXT NOT NULL,
    dob DATE NOT NULL,
    nationality TEXT NOT NULL,
    url TEXT NOT NULL
);

CREATE TABLE seasons (
    year INTEGER PRIMARY KEY,
    url TEXT NOT NULL
);

CREATE TABLE status (
    statusId TEXT PRIMARY KEY,
    status TEXT NOT NULL
);

CREATE TABLE races (
    raceId TEXT PRIMARY KEY,
    year INTEGER NOT NULL,
    round INTEGER NOT NULL,
    circuitId TEXT REFERENCES circuits(circuitId),
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

CREATE TABLE results (
    resultId TEXT PRIMARY KEY,
    raceId TEXT REFERENCES races(raceId),
    driverId TEXT REFERENCES drivers(driverId),
    constructorId TEXT REFERENCES constructors(constructorId),
    number INTEGER,
    grid INTEGER NOT NULL,
    position INTEGER,
    positionText TEXT NOT NULL,
    positionOrder INTEGER NOT NULL,
    points FLOAT NOT NULL,
    laps INTEGER NOT NULL,
    time TEXT,
    milliseconds INTEGER,
    fastestLap INTEGER,
    rank INTEGER,
    fastestLapTime TEXT,
    fastestLapSpeed TEXT,
    statusId TEXT NOT NULL REFERENCES status(statusId)
);