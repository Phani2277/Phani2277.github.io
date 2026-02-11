package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type AuthRequest struct {
	InitData string `json:"initData"`
}

func main() {
	mux := http.NewServeMux()

	if os.Getenv("BOT_TOKEN") == "" {
		log.Fatal("BOT_TOKEN is not set")
	}
	initDB()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
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
	log.Fatal(http.ListenAndServe(":8080", mux))
}
