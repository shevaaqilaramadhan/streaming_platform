package main

import (
	"encoding/json"
	"log"
	"net/http"
	"watchparty-backend/handlers"
	"watchparty-backend/services"
)

func main() {
	http.HandleFunc("/api/rooms", corsMiddleware(handleCreateRoom))
	http.HandleFunc("/ws/", handlers.HandleWebSocket)

	log.Println("🚀 WatchParty server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// corsMiddleware allows any localhost origin regardless of port (5173, 5174, etc.)
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Allow any localhost origin for development
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	roomID := services.CreateRoom()

	response := map[string]string{"roomId": roomID}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println("Failed to encode response:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
