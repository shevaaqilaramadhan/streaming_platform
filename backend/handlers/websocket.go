package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"watchparty-backend/models"
	"watchparty-backend/services"
	"watchparty-backend/utils"

	"github.com/gorilla/websocket"
)

const (
	wsMaxMessageBytes = 64 * 1024
	wsWriteWait       = 10 * time.Second
	wsPongWait        = 60 * time.Second
	wsPingPeriod      = 50 * time.Second
	wsMaxRoomIDLen    = 64
	wsMaxUsernameLen  = 32
)

var (
	roomIDPattern   = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\- .]+$`)

	// BE-L03: per-IP limit on WS upgrades (~30/min, burst 10) to blunt reconnect spam.
	wsUpgradeRateLimit = utils.NewTokenBucket(30.0/60.0, 10)
	// BE-L02: per-IP limit on auto CreateRoomWithID via WS (~6/min, burst 3).
	wsCreateRoomRateLimit = utils.NewTokenBucket(6.0/60.0, 3)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if utils.IsOriginAllowed(origin) {
		return true
	}
	// Vite may rewrite WS Origin to the backend host (e.g. http://localhost:8080).
	if origin != "" && r.Host != "" {
		if host := originHost(origin); host != "" && strings.EqualFold(host, r.Host) {
			return true
		}
	}
	log.Printf("WebSocket origin rejected: %q (Host=%q)", origin, r.Host)
	return false
}

func originHost(origin string) string {
	if i := strings.Index(origin, "://"); i >= 0 {
		origin = origin[i+3:]
	}
	if origin == "" {
		return ""
	}
	if i := strings.IndexAny(origin, "/?"); i >= 0 {
		origin = origin[:i]
	}
	return origin
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/")
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}
	if len(roomID) > wsMaxRoomIDLen || !roomIDPattern.MatchString(roomID) {
		http.Error(w, "Invalid room ID", http.StatusBadRequest)
		return
	}

	clientIP := utils.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP"))
	if !wsUpgradeRateLimit.Allow(clientIP) {
		log.Printf("WS upgrade rate limited for IP %s", clientIP)
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}

	room, exists := services.GetRoom(roomID)
	if !exists {
		// Legacy direct-link path: auto-create room. Rate-limit to reduce spam.
		if !wsCreateRoomRateLimit.Allow(clientIP) {
			log.Printf("WS CreateRoomWithID rate limited for IP %s room %s", clientIP, roomID)
			http.Error(w, "Too many room creations", http.StatusTooManyRequests)
			return
		}
		services.CreateRoomWithID(roomID)
		room, _ = services.GetRoom(roomID)
	}

	if services.RoomClientCount(roomID) >= services.MaxUsersPerRoom {
		http.Error(w, "Room is full", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade failed:", err)
		return
	}

	userID := utils.GenerateShortID()

	user := &models.User{
		ID:   userID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(user.Send)
			_ = user.Conn.Close()
		})
	}

	go writePump(user, cleanup)
	go readPump(user, room, cleanup)
}

func readPump(user *models.User, room *models.WatchRoom, cleanup func()) {
	defer func() {
		// Order: remove from map → close(Send) → Conn.Close()
		// so BroadcastToRoom never sends on a closed channel for this user.
		// RemoveUserFromRoom is idempotent (safe under reconnect/double-cleanup).
		username := user.Username
		userID := user.ID
		services.RemoveUserFromRoom(room.RoomID, user.ID)
		cleanup()

		// Only announce leave if the user had completed JOIN (username set).
		if username != "" {
			leftMsg := models.Message{
				Action: "USER_LEFT",
				Payload: mustMarshal(models.UserLeftPayload{
					UserID:   userID,
					Username: username,
				}),
			}
			services.BroadcastToRoom(room, mustMarshal(leftMsg), nil)
		}
	}()

	user.Conn.SetReadLimit(wsMaxMessageBytes)
	_ = user.Conn.SetReadDeadline(time.Now().Add(wsPongWait))
	user.Conn.SetPongHandler(func(string) error {
		_ = user.Conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	for {
		_, msgBytes, err := user.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		services.HandleMessage(user, room, msgBytes)
	}
}

func writePump(user *models.User, cleanup func()) {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		cleanup()
	}()

	for {
		select {
		case message, ok := <-user.Send:
			_ = user.Conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				_ = user.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := user.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Write error:", err)
				return
			}
		case <-ticker.C:
			_ = user.Conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := user.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ValidateUsername(username string) bool {
	username = strings.TrimSpace(username)
	if username == "" || utf8.RuneCountInString(username) > wsMaxUsernameLen {
		return false
	}
	return usernamePattern.MatchString(username)
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("Marshal error:", err)
		return json.RawMessage("{}")
	}
	return data
}
