package services

import (
	"sync"
	"watchparty-backend/models"
	"watchparty-backend/utils"

	"github.com/gorilla/websocket"
)

var (
	rooms = make(map[string]*models.WatchRoom)
	mu    sync.RWMutex
)

func CreateRoom() string {
	mu.Lock()
	defer mu.Unlock()

	roomID := utils.GenerateShortID()

	for {
		if _, exists := rooms[roomID]; !exists {
			break
		}
		roomID = utils.GenerateShortID()
	}

	rooms[roomID] = newRoom(roomID)
	return roomID
}

// CreateRoomWithID creates a room with a specific ID (used when WS connects before POST /api/rooms).
func CreateRoomWithID(roomID string) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := rooms[roomID]; !exists {
		rooms[roomID] = newRoom(roomID)
	}
}

func newRoom(roomID string) *models.WatchRoom {
	return &models.WatchRoom{
		RoomID:       roomID,
		HostID:       "",
		CurrentVideo: "",
		CurrentTime:  0,
		IsPlaying:    false,
		Clients:      make(map[string]*models.User),
	}
}


func GetRoom(roomID string) (*models.WatchRoom, bool) {
	mu.RLock()
	defer mu.RUnlock()

	room, exists := rooms[roomID]
	return room, exists
}

func AddUserToRoom(roomID, userID, username string, isHost bool, conn *websocket.Conn) (*models.User, error) {
	mu.Lock()
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		return nil, nil
	}

	user := &models.User{
		ID:       userID,
		Username: username,
		IsHost:   isHost,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	room.Mutex.Lock()
	room.Clients[userID] = user
	if isHost && room.HostID == "" {
		room.HostID = userID
	}
	room.Mutex.Unlock()

	return user, nil
}

func RemoveUserFromRoom(roomID, userID string) {
	mu.Lock()
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		return
	}

	room.Mutex.Lock()
	delete(room.Clients, userID)
	clientCount := len(room.Clients)
	room.Mutex.Unlock()

	if clientCount == 0 {
		delete(rooms, roomID)
	}
}

func UpdateRoomState(roomID string, currentTime float64, isPlaying bool) {
	mu.RLock()
	defer mu.RUnlock()

	room, exists := rooms[roomID]
	if !exists {
		return
	}

	room.Mutex.Lock()
	room.CurrentTime = currentTime
	room.IsPlaying = isPlaying
	room.Mutex.Unlock()
}

func SetRoomVideo(roomID, videoURL string) {
	mu.RLock()
	defer mu.RUnlock()

	room, exists := rooms[roomID]
	if !exists {
		return
	}

	room.Mutex.Lock()
	room.CurrentVideo = videoURL
	room.CurrentTime = 0
	room.IsPlaying = false
	room.Mutex.Unlock()
}

func GetRoomState(roomID string) (string, float64, bool, []models.Participant) {
	mu.RLock()
	defer mu.RUnlock()

	room, exists := rooms[roomID]
	if !exists {
		return "", 0, false, nil
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	participants := make([]models.Participant, 0, len(room.Clients))
	for _, client := range room.Clients {
		participants = append(participants, models.Participant{
			UserID:   client.ID,
			Username: client.Username,
		})
	}

	return room.CurrentVideo, room.CurrentTime, room.IsPlaying, participants
}
