package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type Trip struct {
	ID             int       `json:"id"`
	DriverID       int64     `json:"driver_id"`
	FromLocation   string    `json:"from_location"`
	ToLocation     string    `json:"to_location"`
	DepartureTime  time.Time `json:"departure_time"`
	SeatsTotal     int       `json:"seats_total"`
	SeatsAvailable int       `json:"seats_available"`
	CreatedAt      time.Time `json:"created_at"`
}

type createTripRequest struct {
	DriverID      int64     `json:"driver_id"`
	FromLocation  string    `json:"from_location"`
	ToLocation    string    `json:"to_location"`
	DepartureTime time.Time `json:"departure_time"`
	SeatsTotal    int       `json:"seats_total"`
}

func tripsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createTripHandler(w, r)
	case http.MethodGet:
		listTripsHandler(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func createTripHandler(w http.ResponseWriter, r *http.Request) {
	var req createTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	req.FromLocation = strings.TrimSpace(req.FromLocation)
	req.ToLocation = strings.TrimSpace(req.ToLocation)

	if req.DriverID == 0 {
		http.Error(w, "driver_id is required", http.StatusBadRequest)
		return
	}
	if req.FromLocation == "" || req.ToLocation == "" {
		http.Error(w, "from_location and to_location are required", http.StatusBadRequest)
		return
	}
	if req.DepartureTime.IsZero() {
		http.Error(w, "departure_time is required (RFC3339)", http.StatusBadRequest)
		return
	}
	if req.SeatsTotal <= 0 {
		http.Error(w, "seats_total must be greater than 0", http.StatusBadRequest)
		return
	}

	var trip Trip
	err := db.QueryRow(
		`INSERT INTO trips (driver_id, from_location, to_location, departure_time, seats_total, seats_available)
		 VALUES ($1, $2, $3, $4, $5, $5)
		 RETURNING id, driver_id, from_location, to_location, departure_time, seats_total, seats_available, created_at`,
		req.DriverID,
		req.FromLocation,
		req.ToLocation,
		req.DepartureTime,
		req.SeatsTotal,
	).Scan(
		&trip.ID,
		&trip.DriverID,
		&trip.FromLocation,
		&trip.ToLocation,
		&trip.DepartureTime,
		&trip.SeatsTotal,
		&trip.SeatsAvailable,
		&trip.CreatedAt,
	)
	if err != nil {
		log.Println("create trip error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, trip)
}

func listTripsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	rows, err := db.Query(
		`SELECT id, driver_id, from_location, to_location, departure_time, seats_total, seats_available, created_at
		 FROM trips
		 ORDER BY departure_time ASC, id ASC`,
	)
	if err != nil {
		log.Println("list trips error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	trips := make([]Trip, 0)
	for rows.Next() {
		var trip Trip
		if err := rows.Scan(
			&trip.ID,
			&trip.DriverID,
			&trip.FromLocation,
			&trip.ToLocation,
			&trip.DepartureTime,
			&trip.SeatsTotal,
			&trip.SeatsAvailable,
			&trip.CreatedAt,
		); err != nil {
			log.Println("scan trip error:", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		trips = append(trips, trip)
	}

	if err := rows.Err(); err != nil {
		log.Println("iterate trips error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trips)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Println("write json error:", err)
	}
}
