package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"watchparty-backend/models"
	"watchparty-backend/services"
)

func GetPublicRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	allRooms := services.GetAllRooms()
	var publicRooms []*models.PublicRoomInfo

	for _, room := range allRooms {
		room.Mutex.RLock()

		isPublic := room.IsPublic
		clientCount := len(room.Clients)
		queueSize := len(room.Queue)
		hostUsername := ""

		if room.HostID != "" {
			if host, exists := room.Clients[room.HostID]; exists {
				hostUsername = host.Username
			}
		}

		var currentMetadata *models.QueueItem
		if room.CurrentMetadata != nil {
			m := *room.CurrentMetadata
			currentMetadata = &m
		}
		roomName := room.RoomName
		roomID := room.RoomID
		hasPIN := room.PIN != ""

		room.Mutex.RUnlock()

		if !isPublic || clientCount < 1 {
			continue
		}

		publicRoom := &models.PublicRoomInfo{
			RoomID:           roomID,
			RoomName:         roomName,
			HostUsername:     hostUsername,
			ParticipantCount: clientCount,
			QueueSize:        queueSize,
			HasPIN:           hasPIN,
			CurrentMetadata:  currentMetadata,
		}

		publicRooms = append(publicRooms, publicRoom)
	}

	sort.Slice(publicRooms, func(i, j int) bool {
		return publicRooms[i].ParticipantCount > publicRooms[j].ParticipantCount
	})

	if publicRooms == nil {
		publicRooms = []*models.PublicRoomInfo{}
	}

	data, err := json.Marshal(publicRooms)
	if err != nil {
		log.Println("Failed to encode public rooms response:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(data); err != nil {
		log.Println("Failed to write public rooms response:", err)
	}

	log.Printf("Returned %d public rooms", len(publicRooms))
}
