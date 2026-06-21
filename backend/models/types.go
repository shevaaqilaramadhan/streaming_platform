package models

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type User struct {
	ID       string
	Username string
	IsHost   bool
	Conn     *websocket.Conn
	Send     chan []byte
}

type WatchRoom struct {
	RoomID          string
	HostID          string
	CurrentVideo    string
	CurrentTime     float64
	IsPlaying       bool
	CurrentMetadata *VideoMetadata
	Clients         map[string]*User
	Mutex           sync.RWMutex
}

type Message struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type JoinPayload struct {
	RoomID   string `json:"roomId"`
	Username string `json:"username"`
	IsHost   bool   `json:"isHost"`
}

type Participant struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type RoomInitPayload struct {
	RoomID       string         `json:"roomId"`
	CurrentVideo string         `json:"currentVideo"`
	CurrentTime  float64        `json:"currentTime"`
	IsPlaying    bool           `json:"isPlaying"`
	Participants []Participant  `json:"participants"`
	Metadata     *VideoMetadata `json:"metadata,omitempty"`
}

type SyncPayload struct {
	RoomID      string  `json:"roomId"`
	PlayerState string  `json:"playerState"`
	CurrentTime float64 `json:"currentTime"`
	SentAt      int64   `json:"sentAt"`
}

type ChatPayload struct {
	RoomID   string `json:"roomId"`
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Text     string `json:"text"`
	SentAt   int64  `json:"sentAt"`
}

type SetVideoPayload struct {
	RoomID string `json:"roomId"`
	URL    string `json:"url"`
}

type UserLeftPayload struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type JoinBroadcastPayload struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type ScrapeErrorPayload struct {
	OriginalURL string `json:"originalUrl"`
	Error       string `json:"error"`
}

type VideoMetadata struct {
	VideoURL     string `json:"videoUrl"`
	Title        string `json:"title,omitempty"`
	Episode      string `json:"episode,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	Source       string `json:"source,omitempty"`
}
