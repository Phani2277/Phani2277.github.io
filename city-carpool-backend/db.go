package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := ensureSchema(); err != nil {
		log.Fatal(err)
	}

	log.Println("Database connected")
}

func ensureSchema() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			telegram_id BIGINT PRIMARY KEY,
			first_name TEXT NOT NULL,
			username TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS trips (
			id SERIAL PRIMARY KEY,
			driver_id BIGINT NOT NULL,
			from_location TEXT NOT NULL,
			to_location TEXT NOT NULL,
			departure_time TIMESTAMP NOT NULL,
			seats_total INT NOT NULL CHECK (seats_total > 0),
			seats_available INT NOT NULL CHECK (seats_available >= 0),
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS trip_bookings (
			id SERIAL PRIMARY KEY,
			trip_id INT NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
			passenger_id BIGINT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (trip_id, passenger_id)
		);

		CREATE TABLE IF NOT EXISTS driver_profiles (
			telegram_id BIGINT PRIMARY KEY REFERENCES users(telegram_id) ON DELETE CASCADE,
			full_name TEXT NOT NULL,
			phone TEXT NOT NULL,
			car_make TEXT NOT NULL,
			car_number TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS passenger_profiles (
			telegram_id BIGINT PRIMARY KEY REFERENCES users(telegram_id) ON DELETE CASCADE,
			full_name TEXT NOT NULL,
			phone TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	return err
}
