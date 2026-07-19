package models

import (
	"encoding/json"
	"sync"
	"time"

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
	HostToken       string // single-use claim secret; never expose over public APIs
	CurrentVideo    string
	CurrentTime     float64
	IsPlaying       bool
	CurrentMetadata *QueueItem
	Queue           []*QueueItem
	IsPublic        bool
	CreatedAt       time.Time
	EmptySince      time.Time
	RoomName        string
	Clients         map[string]*User
	Mutex           sync.RWMutex
}

type Message struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type JoinPayload struct {
	RoomID    string `json:"roomId"`
	Username  string `json:"username"`
	HostToken string `json:"hostToken,omitempty"`
}

type Participant struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

type RoomInitPayload struct {
	RoomID       string        `json:"roomId"`
	RoomName     string        `json:"roomName,omitempty"`
	CurrentVideo string        `json:"currentVideo"`
	CurrentTime  float64       `json:"currentTime"`
	IsPlaying    bool          `json:"isPlaying"`
	Participants []Participant `json:"participants"`
	Metadata     *QueueItem    `json:"metadata,omitempty"`
	Queue        []*QueueItem  `json:"queue,omitempty"`
	IsPublic     bool          `json:"isPublic"`
	HostID       string        `json:"hostId,omitempty"`
	IsHost       bool          `json:"isHost"`
}

type SyncPayload struct {
	RoomID      string  `json:"roomId"`
	PlayerState string  `json:"playerState"`
	CurrentTime float64 `json:"currentTime"`
	Duration    float64 `json:"duration"`
	SentAt      int64   `json:"sentAt"`
	ServerAt    int64   `json:"serverAt"`
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

type QueueItem struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Episode   string `json:"episode"`
	Thumbnail string `json:"thumbnail"`
}

type AddToQueuePayload struct {
	RoomID string `json:"roomId"`
	URL    string `json:"url"`
}

type RemoveFromQueuePayload struct {
	RoomID string `json:"roomId"`
	ItemID string `json:"itemId"`
}

type QueueUpdatePayload struct {
	RoomID string       `json:"roomId"`
	Queue  []*QueueItem `json:"queue"`
}

type PublicRoomInfo struct {
	RoomID           string     `json:"roomId"`
	RoomName         string     `json:"roomName"`
	HostUsername     string     `json:"hostUsername"`
	ParticipantCount int        `json:"participantCount"`
	QueueSize        int        `json:"queueSize"`
	CurrentMetadata  *QueueItem `json:"currentMetadata,omitempty"`
}

type TogglePublicPayload struct {
	RoomID   string `json:"roomId"`
	IsPublic bool   `json:"isPublic"`
}

type TransferHostPayload struct {
	RoomID       string `json:"roomId"`
	TargetUserID string `json:"targetUserId"`
}

type KickUserPayload struct {
	RoomID       string `json:"roomId"`
	TargetUserID string `json:"targetUserId"`
}

type SetRoomNamePayload struct {
	RoomID   string `json:"roomId"`
	RoomName string `json:"roomName"`
}

type RoomNameChangedPayload struct {
	RoomID   string `json:"roomId"`
	RoomName string `json:"roomName"`
}

type HostChangedPayload struct {
	RoomID    string `json:"roomId"`
	NewHostID string `json:"newHostId"`
	Username  string `json:"username"`
}

type KickedPayload struct {
	RoomID string `json:"roomId"`
	Reason string `json:"reason,omitempty"`
}

type ScrapeStartedPayload struct {
	OriginalURL string `json:"originalUrl"`
	Message     string `json:"message,omitempty"`
}

type ReactionPayload struct {
	RoomID string `json:"roomId"`
	Emoji  string `json:"emoji"`
}

type ReactionBroadcastPayload struct {
	RoomID   string `json:"roomId"`
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Emoji    string `json:"emoji"`
}

type TypingPayload struct {
	RoomID string `json:"roomId"`
}

type TypingBroadcastPayload struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}
