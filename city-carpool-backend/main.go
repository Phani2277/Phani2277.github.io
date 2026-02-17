package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type AuthRequest struct {
	InitData string `json:"initData"`
}

func main() {
	loadEnv()
	mux := http.NewServeMux()

	if os.Getenv("BOT_TOKEN") == "" {
		log.Fatal("BOT_TOKEN is not set")
	}
	initDB()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("city-carpool-backend"))
	})

	mux.HandleFunc("/auth/telegram", func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		json.NewDecoder(r.Body).Decode(&req)

		if !checkTelegramAuth(req.InitData) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := extractTelegramUser(req.InitData)
		if err != nil || user == nil {
			http.Error(w, "bad user data", http.StatusBadRequest)
			return
		}

		if _, err := db.Exec(
			`INSERT INTO users (telegram_id, first_name, username)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (telegram_id) DO NOTHING`,
			user.ID,
			user.FirstName,
			user.Username,
		); err != nil {
			log.Println("db insert error:", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"user":   user,
		})
	})

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}

func loadEnv() {
	// Support running from either backend dir or repo root.
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
