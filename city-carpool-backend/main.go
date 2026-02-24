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
			log.Println("❌ Telegram auth failed")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := extractTelegramUser(req.InitData)
		if err != nil || user == nil {
			log.Println("❌ Bad user data:", err)
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
			log.Println("❌ db insert error:", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		log.Printf("✅ User authenticated: %s (ID: %d)", user.FirstName, user.ID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"user":   user,
		})
	})

	mux.HandleFunc("/trips", tripsHandler)
	mux.HandleFunc("/trips/", tripActionsHandler)
	mux.HandleFunc("/my/bookings", myBookingsHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on :" + port)
	log.Println("📊 Database connected")
	log.Fatal(http.ListenAndServe(":"+port, withRequestLog(withCORS(mux))))
}

func loadEnv() {
	// Support running from either backend dir or repo root.
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ВАЖНО: для ngrok нужны эти заголовки
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, ngrok-skip-browser-warning")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 %s %s (Origin: %s)", r.Method, r.URL.Path, r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
}
