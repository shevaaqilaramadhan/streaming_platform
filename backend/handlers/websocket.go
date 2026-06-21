package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"watchparty-backend/models"
	"watchparty-backend/services"
	"watchparty-backend/utils"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/")
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	room, exists := services.GetRoom(roomID)
	if !exists {
		// Room wasn't pre-created via POST /api/rooms (e.g. host used a direct URL).
		// Auto-create it so the connection can proceed.
		services.CreateRoomWithID(roomID)
		room, _ = services.GetRoom(roomID)
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

	go writePump(user)
	go readPump(user, room)
}

func readPump(user *models.User, room *models.WatchRoom) {
	defer func() {
		user.Conn.Close()
		services.RemoveUserFromRoom(room.RoomID, user.ID)

		if user.Username != "" {
			leftMsg := models.Message{
				Action: "USER_LEFT",
				Payload: mustMarshal(models.UserLeftPayload{
					UserID:   user.ID,
					Username: user.Username,
				}),
			}
			services.BroadcastToRoom(room, mustMarshal(leftMsg), nil)
		}
	}()

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

func writePump(user *models.User) {
	defer user.Conn.Close()

	for message := range user.Send {
		if err := user.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("Marshal error:", err)
		return json.RawMessage("{}")
	}
	return data
}
