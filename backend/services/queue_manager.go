package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"
	"watchparty-backend/models"
)

func generateQueueItemID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("q_%d", time.Now().UnixNano())
	}
	return "q_" + hex.EncodeToString(bytes)
}

func AddToQueue(roomID, url string) (*models.QueueItem, error) {
	room, exists := GetRoom(roomID)
	if !exists {
		return nil, fmt.Errorf("room not found")
	}

	log.Printf("Adding to queue in room %s: %s", roomID, url)

	var queueItem *models.QueueItem

	if isDirectStreamURL(url) || isYouTubeURL(url) {
		// Raw stream/YouTube URL pasted directly — no page to scrape.
		queueItem = &models.QueueItem{
			ID:        generateQueueItemID(),
			URL:       url,
			Title:     extractDomainFromURL(url),
			Episode:   "",
			Thumbnail: "",
		}
	} else {
		metadata, err := ScrapeStreamURLCached(url)
		if err != nil {
			log.Printf("Scrape failed for queue item %s: %v", url, err)
			return nil, fmt.Errorf("failed to scrape metadata: %w", err)
		}

		queueItem = &models.QueueItem{
			ID:        generateQueueItemID(),
			URL:       metadata.VideoURL,
			Title:     metadata.Title,
			Episode:   metadata.Episode,
			Thumbnail: metadata.ThumbnailURL,
		}
	}

	room.Mutex.Lock()
	room.Queue = append(room.Queue, queueItem)
	room.Mutex.Unlock()

	log.Printf("Added to queue: %s - %s (ID: %s)", queueItem.Title, queueItem.Episode, queueItem.ID)

	return queueItem, nil
}

func RemoveFromQueue(roomID, itemID string) error {
	room, exists := GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found")
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	for i, item := range room.Queue {
		if item.ID == itemID {
			room.Queue = append(room.Queue[:i], room.Queue[i+1:]...)
			log.Printf("Removed from queue: %s (ID: %s)", item.Title, itemID)
			return nil
		}
	}

	return fmt.Errorf("queue item not found")
}

func SkipToNext(roomID string) (*models.QueueItem, error) {
	return AdvanceQueue(roomID)
}

func AdvanceQueue(roomID string) (*models.QueueItem, error) {
	room, exists := GetRoom(roomID)
	if !exists {
		return nil, fmt.Errorf("room not found")
	}

	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	if len(room.Queue) == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	nextItem := room.Queue[0]
	room.Queue = room.Queue[1:]

	copied := *nextItem
	room.CurrentVideo = copied.URL
	room.CurrentMetadata = &models.QueueItem{
		ID:        copied.ID,
		URL:       copied.URL,
		Title:     copied.Title,
		Episode:   copied.Episode,
		Thumbnail: copied.Thumbnail,
	}
	room.CurrentTime = 0
	room.IsPlaying = false

	log.Printf("Advancing queue: %s - %s (ID: %s)", copied.Title, copied.Episode, copied.ID)

	return &copied, nil
}

func ClearQueue(roomID string) error {
	room, exists := GetRoom(roomID)
	if !exists {
		return fmt.Errorf("room not found")
	}

	room.Mutex.Lock()
	room.Queue = []*models.QueueItem{}
	room.Mutex.Unlock()

	log.Printf("Queue cleared for room %s", roomID)

	return nil
}

func GetQueue(roomID string) []*models.QueueItem {
	room, exists := GetRoom(roomID)
	if !exists {
		return []*models.QueueItem{}
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	queueCopy := make([]*models.QueueItem, len(room.Queue))
	for i, item := range room.Queue {
		if item != nil {
			copied := *item
			queueCopy[i] = &copied
		}
	}

	return queueCopy
}
