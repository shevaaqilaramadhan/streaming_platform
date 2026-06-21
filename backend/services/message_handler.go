package services

import (
	"encoding/json"
	"log"
	"watchparty-backend/models"
)

func HandleMessage(user *models.User, room *models.WatchRoom, msgBytes []byte) {
	var msg models.Message
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		log.Println("Invalid JSON:", err)
		return
	}

	switch msg.Action {
	case "JOIN_EVENT":
		handleJoinEvent(user, room, msg.Payload)
	case "SYNC_EVENT":
		handleSyncEvent(user, room, msg.Payload)
	case "CHAT_EVENT":
		handleChatEvent(user, room, msg.Payload)
	case "SET_VIDEO":
		handleSetVideo(user, room, msg.Payload)
	default:
		log.Println("Unknown action:", msg.Action)
	}
}

func handleJoinEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	var payload models.JoinPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid JOIN_EVENT payload:", err)
		return
	}

	user.Username = payload.Username
	user.IsHost = payload.IsHost

	// Register the user to the room's clients map so they receive broadcasts and appear in participants list
	room.Mutex.Lock()
	room.Clients[user.ID] = user
	if user.IsHost && room.HostID == "" {
		room.HostID = user.ID
	}
	room.Mutex.Unlock()

	currentVideo, currentTime, isPlaying, participants := GetRoomState(room.RoomID)

	roomInitMsg := models.Message{
		Action: "ROOM_INIT",
		Payload: json.RawMessage(mustMarshal(models.RoomInitPayload{
			RoomID:       room.RoomID,
			CurrentVideo: currentVideo,
			CurrentTime:  currentTime,
			IsPlaying:    isPlaying,
			Participants: participants,
		})),
	}

	SendToUser(user, mustMarshal(roomInitMsg))

	joinBroadcastMsg := models.Message{
		Action: "JOIN_EVENT",
		Payload: json.RawMessage(mustMarshal(models.JoinBroadcastPayload{
			UserID:   user.ID,
			Username: user.Username,
		})),
	}

	BroadcastToRoom(room, mustMarshal(joinBroadcastMsg), user)
}

func handleSyncEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	var payload models.SyncPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid SYNC_EVENT payload:", err)
		return
	}

	isPlaying := payload.PlayerState == "PLAYING"
	UpdateRoomState(room.RoomID, payload.CurrentTime, isPlaying)

	msg := models.Message{
		Action:  "SYNC_EVENT",
		Payload: payloadRaw,
	}

	BroadcastToRoom(room, mustMarshal(msg), user)
}

func handleChatEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	var payload models.ChatPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid CHAT_EVENT payload:", err)
		return
	}

	msg := models.Message{
		Action:  "CHAT_EVENT",
		Payload: payloadRaw,
	}

	BroadcastToRoom(room, mustMarshal(msg), nil)
}

func handleSetVideo(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	var payload models.SetVideoPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid SET_VIDEO payload:", err)
		return
	}

	SetRoomVideo(room.RoomID, payload.URL)

	msg := models.Message{
		Action:  "SET_VIDEO",
		Payload: payloadRaw,
	}

	BroadcastToRoom(room, mustMarshal(msg), nil)
}

func BroadcastToRoom(room *models.WatchRoom, message []byte, exclude *models.User) {
	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	for _, client := range room.Clients {
		if exclude != nil && client.ID == exclude.ID {
			continue
		}
		select {
		case client.Send <- message:
		default:
			log.Println("Client send buffer full:", client.ID)
		}
	}
}

func SendToUser(user *models.User, message []byte) {
	select {
	case user.Send <- message:
	default:
		log.Println("User send buffer full:", user.ID)
	}
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("Marshal error:", err)
		return []byte("{}")
	}
	return data
}
