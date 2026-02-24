package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
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

type bookTripRequest struct {
	PassengerID int64 `json:"passenger_id"`
}

type MyBooking struct {
	BookingID      int       `json:"booking_id"`
	BookedAt       time.Time `json:"booked_at"`
	TripID         int       `json:"trip_id"`
	DriverID       int64     `json:"driver_id"`
	FromLocation   string    `json:"from_location"`
	ToLocation     string    `json:"to_location"`
	DepartureTime  time.Time `json:"departure_time"`
	SeatsTotal     int       `json:"seats_total"`
	SeatsAvailable int       `json:"seats_available"`
	TripCreatedAt  time.Time `json:"trip_created_at"`
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

func tripActionsHandler(w http.ResponseWriter, r *http.Request) {
	tripID, ok := parseTripBookPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bookTripHandler(w, r, tripID)
}

func parseTripBookPath(path string) (int, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "trips" || parts[2] != "book" {
		return 0, false
	}

	tripID, err := strconv.Atoi(parts[1])
	if err != nil || tripID <= 0 {
		return 0, false
	}

	return tripID, true
}

func createTripHandler(w http.ResponseWriter, r *http.Request) {
	var req createTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("decode error:", err)
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

	log.Printf("✅ Created trip: ID=%d, %s -> %s", trip.ID, trip.FromLocation, trip.ToLocation)
	writeJSON(w, http.StatusCreated, trip)
}

func listTripsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("📋 GET /trips - fetching trips...")

	w.Header().Set("Cache-Control", "no-store")

	// Сначала проверим сколько записей в БД
	var count int
	db.QueryRow("SELECT COUNT(*) FROM trips").Scan(&count)
	log.Printf("📊 Total trips in DB: %d", count)

	rows, err := db.Query(
		`SELECT id, driver_id, from_location, to_location, departure_time, seats_total, seats_available, created_at
		 FROM trips
		 ORDER BY created_at DESC, id DESC`,
	)
	if err != nil {
		log.Println("❌ list trips query error:", err)
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
			log.Println("❌ scan trip error:", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		trips = append(trips, trip)
	}

	if err := rows.Err(); err != nil {
		log.Println("❌ iterate trips error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Returning %d trips", len(trips))
	writeJSON(w, http.StatusOK, trips)
}

func myBookingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	passengerIDParam := strings.TrimSpace(r.URL.Query().Get("passenger_id"))
	if passengerIDParam == "" {
		http.Error(w, "passenger_id is required", http.StatusBadRequest)
		return
	}

	passengerID, err := strconv.ParseInt(passengerIDParam, 10, 64)
	if err != nil || passengerID == 0 {
		http.Error(w, "invalid passenger_id", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(
		`SELECT
			b.id,
			b.created_at,
			t.id,
			t.driver_id,
			t.from_location,
			t.to_location,
			t.departure_time,
			t.seats_total,
			t.seats_available,
			t.created_at
		FROM trip_bookings b
		JOIN trips t ON t.id = b.trip_id
		WHERE b.passenger_id = $1
		ORDER BY b.created_at DESC, b.id DESC`,
		passengerID,
	)
	if err != nil {
		log.Println("list my bookings query error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	bookings := make([]MyBooking, 0)
	for rows.Next() {
		var booking MyBooking
		if err := rows.Scan(
			&booking.BookingID,
			&booking.BookedAt,
			&booking.TripID,
			&booking.DriverID,
			&booking.FromLocation,
			&booking.ToLocation,
			&booking.DepartureTime,
			&booking.SeatsTotal,
			&booking.SeatsAvailable,
			&booking.TripCreatedAt,
		); err != nil {
			log.Println("scan my booking error:", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		log.Println("iterate my bookings error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, bookings)
}

func bookTripHandler(w http.ResponseWriter, r *http.Request, tripID int) {
	var req bookTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if req.PassengerID == 0 {
		http.Error(w, "passenger_id is required", http.StatusBadRequest)
		return
	}

	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println("begin tx error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var driverID int64
	var seatsAvailable int
	err = tx.QueryRowContext(
		r.Context(),
		`SELECT driver_id, seats_available FROM trips WHERE id = $1 FOR UPDATE`,
		tripID,
	).Scan(&driverID, &seatsAvailable)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		log.Println("trip lookup error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if req.PassengerID == driverID {
		http.Error(w, "driver cannot book own trip", http.StatusBadRequest)
		return
	}
	if seatsAvailable <= 0 {
		http.Error(w, "no seats available", http.StatusConflict)
		return
	}

	_, err = tx.ExecContext(
		r.Context(),
		`INSERT INTO trip_bookings (trip_id, passenger_id) VALUES ($1, $2)`,
		tripID,
		req.PassengerID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "already booked", http.StatusConflict)
			return
		}
		log.Println("create booking error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	result, err := tx.ExecContext(
		r.Context(),
		`UPDATE trips
		 SET seats_available = seats_available - 1
		 WHERE id = $1 AND seats_available > 0`,
		tripID,
	)
	if err != nil {
		log.Println("update seats error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("rows affected error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "no seats available", http.StatusConflict)
		return
	}

	var trip Trip
	err = tx.QueryRowContext(
		r.Context(),
		`SELECT id, driver_id, from_location, to_location, departure_time, seats_total, seats_available, created_at
		 FROM trips
		 WHERE id = $1`,
		tripID,
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
		log.Println("fetch booked trip error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("commit booking error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "booked",
		"trip":   trip,
	})
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == "23505"
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Println("❌ write json error:", err)
	}
}
