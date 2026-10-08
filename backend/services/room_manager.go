package services

import (
	"log"
	"strings"
	"sync"
	"time"
	"watchparty-backend/models"
	"watchparty-backend/utils"
)

const (
	MaxUsersPerRoom      = 20
	roomEmptyGracePeriod = 3 * time.Minute
	staleRoomMaxAge      = 30 * time.Minute
)

var (
	rooms = make(map[string]*models.WatchRoom)
	mu    sync.RWMutex
)

// CreateRoom creates a room and returns (roomID, hostToken).
// hostToken is single-use: first valid JOIN_EVENT with it becomes host.
func CreateRoom() (roomID, hostToken string) {
	return CreateRoomWithPIN("")
}

// CreateRoomWithPIN creates a room with an optional access PIN.
func CreateRoomWithPIN(pin string) (roomID, hostToken string) {
	mu.Lock()
	defer mu.Unlock()

	roomID = utils.GenerateShortID()
	for {
		if _, exists := rooms[roomID]; !exists {
			break
		}
		roomID = utils.GenerateShortID()
	}

	hostToken = utils.GenerateHostToken()
	room := newRoom(roomID)
	room.HostToken = hostToken
	room.PIN = strings.TrimSpace(pin)
	rooms[roomID] = room
	return roomID, hostToken
}

// CreateRoomWithID creates a room with a specific ID (WS join before POST /api/rooms).
// No host token — first joiner becomes host (legacy / direct-link path).
func CreateRoomWithID(roomID string) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := rooms[roomID]; !exists {
		rooms[roomID] = newRoom(roomID)
	}
}

func newRoom(roomID string) *models.WatchRoom {
	return &models.WatchRoom{
		RoomID:          roomID,
		HostID:          "",
		HostToken:       "",
		CurrentVideo:    "",
		CurrentTime:     0,
		IsPlaying:       false,
		CurrentMetadata: nil,
		Queue:           []*models.QueueItem{},
		IsPublic:        false,
		CreatedAt:       time.Now(),
		RoomName:        "",
		Clients:         make(map[string]*models.User),
	}
}

func GetRoom(roomID string) (*models.WatchRoom, bool) {
	mu.RLock()
	defer mu.RUnlock()

	room, exists := rooms[roomID]
	return room, exists
}

func GetAllRooms() []*models.WatchRoom {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]*models.WatchRoom, 0, len(rooms))
	for _, room := range rooms {
		result = append(result, room)
	}
	return result
}

func RoomClientCount(roomID string) int {
	room, exists := GetRoom(roomID)
	if !exists {
		return 0
	}
	room.Mutex.RLock()
	defer room.Mutex.RUnlock()
	return len(room.Clients)
}

// StartRoomCleanup periodically removes rooms that have been empty past the
// grace period, and stale empty rooms older than staleRoomMaxAge.
// Lock order: always global mu before room.Mutex.
func StartRoomCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			mu.RLock()
			var staleIDs []string
			now := time.Now()
			for id, room := range rooms {
				room.Mutex.RLock()
				isEmpty := len(room.Clients) == 0
				emptySince := room.EmptySince
				age := now.Sub(room.CreatedAt)
				room.Mutex.RUnlock()

				if !isEmpty {
					continue
				}
				// Grace period after last client left, or long-lived never-joined rooms.
				if (!emptySince.IsZero() && now.Sub(emptySince) > roomEmptyGracePeriod) ||
					(emptySince.IsZero() && age > staleRoomMaxAge) {
					staleIDs = append(staleIDs, id)
				}
			}
			mu.RUnlock()

			if len(staleIDs) == 0 {
				continue
			}

			mu.Lock()
			for _, id := range staleIDs {
				if r, ok := rooms[id]; ok {
					r.Mutex.RLock()
					stillEmpty := len(r.Clients) == 0
					r.Mutex.RUnlock()
					if stillEmpty {
						delete(rooms, id)
						PruneEpisodeEnded(id)
						log.Printf("Cleaned up stale room: %s", id)
					}
				}
			}
			mu.Unlock()
		}
	}()
}

func RemoveUserFromRoom(roomID, userID string) {
	// Lock order: always global mu before room.Mutex
	mu.RLock()
	room, exists := rooms[roomID]
	mu.RUnlock()
	if !exists {
		return
	}

	var newHostID, newHostUsername string

	room.Mutex.Lock()
	// Idempotent: user may already be gone (double cleanup / reconnect race).
	if _, ok := room.Clients[userID]; !ok {
		// Still fix orphan host pointer if HostID points at a missing client.
		if room.HostID == userID {
			room.HostID = ""
			for _, client := range room.Clients {
				room.HostID = client.ID
				client.IsHost = true
				newHostID = client.ID
				newHostUsername = client.Username
				log.Printf("Host missing in room %s. New host: %s (%s)", roomID, client.Username, client.ID)
				break
			}
		}
		clientCount := len(room.Clients)
		if clientCount == 0 {
			if room.EmptySince.IsZero() {
				room.EmptySince = time.Now()
			}
		} else {
			room.EmptySince = time.Time{}
		}
		room.Mutex.Unlock()
		if newHostID != "" {
			broadcastHostChanged(room, roomID, newHostID, newHostUsername)
		}
		return
	}

	delete(room.Clients, userID)

	// Host reassignment: if the leaving user was host, promote next client
	if room.HostID == userID {
		room.HostID = ""
		for _, client := range room.Clients {
			room.HostID = client.ID
			client.IsHost = true
			newHostID = client.ID
			newHostUsername = client.Username
			log.Printf("Host left room %s. New host: %s (%s)", roomID, client.Username, client.ID)
			break
		}
	}

	clientCount := len(room.Clients)
	if clientCount == 0 {
		room.EmptySince = time.Now()
	} else {
		room.EmptySince = time.Time{}
	}
	room.Mutex.Unlock()

	if newHostID != "" {
		broadcastHostChanged(room, roomID, newHostID, newHostUsername)
	}

	// Room empty: do NOT delete immediately — grace period via StartRoomCleanup.
}

func broadcastHostChanged(room *models.WatchRoom, roomID, newHostID, newHostUsername string) {
	hostChangedMsg := models.Message{
		Action: "HOST_CHANGED",
		Payload: mustMarshalRaw(models.HostChangedPayload{
			RoomID:    roomID,
			NewHostID: newHostID,
			Username:  newHostUsername,
		}),
	}
	BroadcastToRoom(room, mustMarshal(hostChangedMsg), nil)
}

func UpdateRoomState(roomID string, currentTime float64, isPlaying bool) {
	room, exists := GetRoom(roomID)
	if !exists {
		return
	}

	room.Mutex.Lock()
	room.CurrentTime = currentTime
	room.IsPlaying = isPlaying
	room.Mutex.Unlock()
}

func SetRoomMetadata(roomID string, metadata *models.VideoMetadata) {
	room, exists := GetRoom(roomID)
	if !exists {
		return
	}

	room.Mutex.Lock()
	room.CurrentVideo = metadata.VideoURL
	room.CurrentMetadata = &models.QueueItem{
		ID:             "",
		URL:            metadata.VideoURL,
		Title:          metadata.Title,
		Episode:        metadata.Episode,
		Thumbnail:      metadata.ThumbnailURL,
		NextEpisodeURL: metadata.NextEpisodeURL,
	}
	room.CurrentTime = 0
	room.IsPlaying = false
	room.Mutex.Unlock()
}

func SetRoomPIN(roomID, pin string) {
	room, exists := GetRoom(roomID)
	if !exists {
		return
	}
	room.Mutex.Lock()
	room.PIN = strings.TrimSpace(pin)
	room.Mutex.Unlock()
}

func GetRoomState(roomID string) (string, float64, bool, []models.Participant, *models.QueueItem, []*models.QueueItem) {
	room, exists := GetRoom(roomID)
	if !exists {
		return "", 0, false, nil, nil, nil
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

	// Deep-copy metadata and queue so callers never hold live pointers
	// after the room lock is released.
	var metadata *models.QueueItem
	if room.CurrentMetadata != nil {
		m := *room.CurrentMetadata
		metadata = &m
	}

	queueCopy := make([]*models.QueueItem, len(room.Queue))
	for i, item := range room.Queue {
		if item != nil {
			copied := *item
			queueCopy[i] = &copied
		}
	}

	return room.CurrentVideo, room.CurrentTime, room.IsPlaying, participants, metadata, queueCopy
}
