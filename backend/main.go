package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"watchparty-backend/handlers"
	"watchparty-backend/services"
	"watchparty-backend/utils"
)

func main() {
	startTime := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		rooms, users := services.GetRoomStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"uptimeSec": int(time.Since(startTime).Seconds()),
			"rooms":     rooms,
			"users":     users,
			"timestamp": time.Now().Unix(),
		})
	}))
	mux.HandleFunc("/api/rooms", corsMiddleware(handleCreateRoom))
	mux.HandleFunc("/api/public-rooms", corsMiddleware(handlers.GetPublicRooms))
	mux.HandleFunc("/api/proxy", corsMiddleware(handlers.HandleStreamProxy))
	mux.HandleFunc("/api/search", corsMiddleware(handlers.HandleSearch))
	mux.HandleFunc("/api/catalog/episodes", corsMiddleware(handlers.HandleCatalogEpisodes))
	mux.HandleFunc("/ws/", corsMiddleware(handlers.HandleWebSocket))

	services.StartRoomCleanup(10 * time.Minute)

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		addr = v
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("WatchParty server running on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && utils.IsOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Range, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

type createRoomReq struct {
	PIN string `json:"pin,omitempty"`
}

func handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
		if roomID == "" {
			http.Error(w, "roomId is required", http.StatusBadRequest)
			return
		}
		room, exists := services.GetRoom(roomID)
		hasPin := false
		isPublic := false
		roomName := ""
		if exists && room != nil {
			room.Mutex.Lock()
			hasPin = room.PIN != ""
			isPublic = room.IsPublic
			roomName = room.RoomName
			room.Mutex.Unlock()
		}
		response := map[string]interface{}{
			"roomId":   roomID,
			"exists":   exists,
			"hasPin":   hasPin,
			"isPublic": isPublic,
			"roomName": roomName,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createRoomReq
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	roomID, hostToken := services.CreateRoomWithPIN(req.PIN)
	response := map[string]interface{}{
		"roomId":    roomID,
		"hostToken": hostToken,
		"hasPin":    req.PIN != "",
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Println("Failed to encode response:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(data); err != nil {
		log.Println("Failed to write response:", err)
	}
}
