package services

import (
	"encoding/json"
	"log"
	neturl "net/url"
	"strings"
	"watchparty-backend/models"
)

func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") ||
		strings.Contains(url, "youtu.be")
}

func isDirectStreamURL(url string) bool {
	lower := strings.ToLower(url)
	parsed, err := neturl.Parse(lower)
	if err != nil {
		return false
	}
	path := parsed.Path
	return strings.HasSuffix(path, ".mp4") ||
		strings.HasSuffix(path, ".m3u8") ||
		strings.HasSuffix(path, ".webm") ||
		strings.HasSuffix(path, ".mkv")
}

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

	finalURL := payload.URL

	if !isYouTubeURL(payload.URL) && !isDirectStreamURL(payload.URL) {
		log.Printf("Scraping anime page: %s", payload.URL)
		scraped, err := ScrapeStreamURL(payload.URL)
		if err != nil {
			log.Printf("Scrape failed for %s: %v", payload.URL, err)
			errMsg := models.Message{
				Action: "SCRAPE_ERROR",
				Payload: mustMarshalRaw(models.ScrapeErrorPayload{
					OriginalURL: payload.URL,
					Error:       err.Error(),
				}),
			}
			SendToUser(user, mustMarshal(errMsg))
			return
		}
		log.Printf("Scrape succeeded, found: %s", scraped)
		finalURL = scraped
	}

	SetRoomVideo(room.RoomID, finalURL)

	broadcastPayload := map[string]string{
		"roomId": payload.RoomID,
		"url":    finalURL,
	}
	msg := models.Message{
		Action:  "SET_VIDEO",
		Payload: mustMarshalRaw(broadcastPayload),
	}
	BroadcastToRoom(room, mustMarshal(msg), nil)
}

func mustMarshalRaw(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("Marshal error:", err)
		return json.RawMessage("{}")
	}
	return data
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
