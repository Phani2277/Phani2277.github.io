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
)

type DriverProfile struct {
	TelegramID int64     `json:"telegram_id"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone"`
	CarMake    string    `json:"car_make"`
	CarNumber  string    `json:"car_number"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PassengerProfile struct {
	TelegramID int64     `json:"telegram_id"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type driverProfileRequest struct {
	TelegramID int64  `json:"telegram_id"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	CarMake    string `json:"car_make"`
	CarNumber  string `json:"car_number"`
}

type passengerProfileRequest struct {
	TelegramID int64  `json:"telegram_id"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
}

func profilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	telegramIDParam := strings.TrimSpace(r.URL.Query().Get("telegram_id"))
	if telegramIDParam == "" {
		http.Error(w, "telegram_id is required", http.StatusBadRequest)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDParam, 10, 64)
	if err != nil || telegramID == 0 {
		http.Error(w, "invalid telegram_id", http.StatusBadRequest)
		return
	}

	driver, err := getDriverProfile(telegramID)
	if err != nil {
		log.Println("get driver profile error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	passenger, err := getPassengerProfile(telegramID)
	if err != nil {
		log.Println("get passenger profile error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"driver":    driver,
		"passenger": passenger,
	})
}

func upsertDriverProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req driverProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.Phone = strings.TrimSpace(req.Phone)
	req.CarMake = strings.TrimSpace(req.CarMake)
	req.CarNumber = strings.TrimSpace(req.CarNumber)

	if req.TelegramID == 0 {
		http.Error(w, "telegram_id is required", http.StatusBadRequest)
		return
	}
	if req.FullName == "" || req.Phone == "" || req.CarMake == "" || req.CarNumber == "" {
		http.Error(w, "full_name, phone, car_make and car_number are required", http.StatusBadRequest)
		return
	}

	var profile DriverProfile
	err := db.QueryRow(
		`INSERT INTO driver_profiles (telegram_id, full_name, phone, car_make, car_number)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (telegram_id) DO UPDATE SET
		   full_name = EXCLUDED.full_name,
		   phone = EXCLUDED.phone,
		   car_make = EXCLUDED.car_make,
		   car_number = EXCLUDED.car_number,
		   updated_at = NOW()
		 RETURNING telegram_id, full_name, phone, car_make, car_number, created_at, updated_at`,
		req.TelegramID,
		req.FullName,
		req.Phone,
		req.CarMake,
		req.CarNumber,
	).Scan(
		&profile.TelegramID,
		&profile.FullName,
		&profile.Phone,
		&profile.CarMake,
		&profile.CarNumber,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		log.Println("upsert driver profile error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func upsertPassengerProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req passengerProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.TelegramID == 0 {
		http.Error(w, "telegram_id is required", http.StatusBadRequest)
		return
	}
	if req.FullName == "" || req.Phone == "" {
		http.Error(w, "full_name and phone are required", http.StatusBadRequest)
		return
	}

	var profile PassengerProfile
	err := db.QueryRow(
		`INSERT INTO passenger_profiles (telegram_id, full_name, phone)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (telegram_id) DO UPDATE SET
		   full_name = EXCLUDED.full_name,
		   phone = EXCLUDED.phone,
		   updated_at = NOW()
		 RETURNING telegram_id, full_name, phone, created_at, updated_at`,
		req.TelegramID,
		req.FullName,
		req.Phone,
	).Scan(
		&profile.TelegramID,
		&profile.FullName,
		&profile.Phone,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		log.Println("upsert passenger profile error:", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func getDriverProfile(telegramID int64) (*DriverProfile, error) {
	var profile DriverProfile
	err := db.QueryRow(
		`SELECT telegram_id, full_name, phone, car_make, car_number, created_at, updated_at
		 FROM driver_profiles
		 WHERE telegram_id = $1`,
		telegramID,
	).Scan(
		&profile.TelegramID,
		&profile.FullName,
		&profile.Phone,
		&profile.CarMake,
		&profile.CarNumber,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func getPassengerProfile(telegramID int64) (*PassengerProfile, error) {
	var profile PassengerProfile
	err := db.QueryRow(
		`SELECT telegram_id, full_name, phone, created_at, updated_at
		 FROM passenger_profiles
		 WHERE telegram_id = $1`,
		telegramID,
	).Scan(
		&profile.TelegramID,
		&profile.FullName,
		&profile.Phone,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}
