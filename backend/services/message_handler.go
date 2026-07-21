package services

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	neturl "net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
	"watchparty-backend/models"
	"watchparty-backend/utils"
)

const (
	maxChatTextLen       = 500
	maxQueueSize         = 50
	maxUsernameLen       = 32
	maxRoomNameLen       = 64
	maxEmojiLen          = 8
	episodeEndedDebounce = 3 * time.Second
)

var (
	lastEpisodeEnded sync.Map // roomID -> *atomic.Int64 (unix nano)
	usernamePattern  = regexp.MustCompile(`^[a-zA-Z0-9_\- .]+$`)
	roomNamePattern  = regexp.MustCompile(`^[\p{L}\p{N}_\- .!?'"&()]+$`)
	urlSchemePattern = regexp.MustCompile(`^https?://`)
	// Common emoji / short reaction tokens (single grapheme or short text).
	allowedReactions = map[string]struct{}{
		"👍": {}, "👎": {}, "❤️": {}, "🔥": {}, "😂": {}, "😮": {}, "😢": {}, "👏": {},
		"🎉": {}, "💯": {}, "👀": {}, "✨": {}, "🤣": {}, "😍": {}, "🤔": {}, "💀": {},
		"+1": {}, "-1": {}, "lol": {}, "wow": {}, "gg": {}, "rip": {},
	}

	// Per-user WS action limits (in-memory token buckets).
	// chat: ~20/min, setVideo/scrape: ~6/min, queue add: ~10/min
	// reaction: ~30/min, typing: ~1 per 2s
	chatRateLimit     = utils.NewTokenBucket(20.0/60.0, 5)
	setVideoRateLimit = utils.NewTokenBucket(6.0/60.0, 2)
	queueRateLimit    = utils.NewTokenBucket(10.0/60.0, 3)
	reactionRateLimit = utils.NewTokenBucket(30.0/60.0, 5)
	typingRateLimit   = utils.NewTokenBucket(0.5, 1)
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

func isValidMediaURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2048 {
		return false
	}
	if !urlSchemePattern.MatchString(raw) {
		return false
	}
	parsed, err := neturl.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func sanitizeUsername(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return "Anonymous"
	}
	if utf8.RuneCountInString(username) > maxUsernameLen {
		runes := []rune(username)
		username = string(runes[:maxUsernameLen])
	}
	if !usernamePattern.MatchString(username) {
		return "Anonymous"
	}
	return username
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
	case "ADD_TO_QUEUE":
		handleAddToQueue(user, room, msg.Payload)
	case "REMOVE_FROM_QUEUE":
		handleRemoveFromQueue(user, room, msg.Payload)
	case "SKIP_TO_NEXT":
		handleSkipToNext(user, room, msg.Payload)
	case "CLEAR_QUEUE":
		handleClearQueue(user, room, msg.Payload)
	case "EPISODE_ENDED":
		handleEpisodeEnded(user, room, msg.Payload)
	case "TOGGLE_PUBLIC":
		handleTogglePublic(user, room, msg.Payload)
	case "TRANSFER_HOST":
		handleTransferHost(user, room, msg.Payload)
	case "KICK_USER":
		handleKickUser(user, room, msg.Payload)
	case "SET_ROOM_NAME":
		handleSetRoomName(user, room, msg.Payload)
	case "REACTION":
		handleReaction(user, room, msg.Payload)
	case "TYPING":
		handleTyping(user, room, msg.Payload)
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

	user.Username = sanitizeUsername(payload.Username)

	room.Mutex.Lock()
	// Idempotent re-JOIN on same connection (frontend may re-send JOIN on reconnect
	// of the logical session, or spam JOIN). Already in map → refresh state only.
	alreadyInRoom := false
	if existing, ok := room.Clients[user.ID]; ok && existing == user {
		alreadyInRoom = true
	}

	if !alreadyInRoom {
		if len(room.Clients) >= MaxUsersPerRoom {
			room.Mutex.Unlock()
			log.Printf("Room %s full, rejecting join from %s", room.RoomID, user.Username)
			rejectMsg := models.Message{
				Action: "JOIN_REJECTED",
				Payload: mustMarshalRaw(map[string]interface{}{
					"roomId": room.RoomID,
					"reason": "room_full",
				}),
			}
			SendToUser(user, mustMarshal(rejectMsg))
			return
		}
		room.Clients[user.ID] = user
	}

	room.EmptySince = time.Time{}
	assignHostOnJoin(user, room, payload.HostToken)
	isPublic := room.IsPublic
	hostID := room.HostID
	isHost := user.IsHost
	roomName := room.RoomName
	room.Mutex.Unlock()

	currentVideo, currentTime, isPlaying, participants, metadata, queue := GetRoomState(room.RoomID)

	roomInitMsg := models.Message{
		Action: "ROOM_INIT",
		Payload: json.RawMessage(mustMarshal(models.RoomInitPayload{
			RoomID:       room.RoomID,
			RoomName:     roomName,
			CurrentVideo: currentVideo,
			CurrentTime:  currentTime,
			IsPlaying:    isPlaying,
			Participants: participants,
			Metadata:     metadata,
			Queue:        queue,
			IsPublic:     isPublic,
			HostID:       hostID,
			IsHost:       isHost,
		})),
	}

	SendToUser(user, mustMarshal(roomInitMsg))

	// Only broadcast presence on first successful join for this user ID.
	if !alreadyInRoom {
		joinBroadcastMsg := models.Message{
			Action: "JOIN_EVENT",
			Payload: json.RawMessage(mustMarshal(models.JoinBroadcastPayload{
				UserID:   user.ID,
				Username: user.Username,
			})),
		}
		BroadcastToRoom(room, mustMarshal(joinBroadcastMsg), user)
	}
}

// assignHostOnJoin sets host privilege. Caller must hold room.Mutex.
// Host claim rules (server-authoritative; client isHost is ignored):
//  1. Already host on this connection → keep
//  2. Host seat taken → guest
//  3. HostToken set (POST /api/rooms) → only matching single-use token claims host
//  4. No token (legacy CreateRoomWithID / empty room after leave) → first joiner is host
func assignHostOnJoin(user *models.User, room *models.WatchRoom, hostToken string) {
	if room.HostID != "" {
		user.IsHost = room.HostID == user.ID
		return
	}

	if room.HostToken != "" {
		if hostTokenMatches(hostToken, room.HostToken) {
			room.HostID = user.ID
			room.HostToken = ""
			user.IsHost = true
			log.Printf("User %s (%s) claimed host via token for room %s", user.Username, user.ID, room.RoomID)
			return
		}
		user.IsHost = false
		return
	}

	room.HostID = user.ID
	user.IsHost = true
	log.Printf("User %s (%s) assigned as host for room %s", user.Username, user.ID, room.RoomID)
}

func hostTokenMatches(provided, expected string) bool {
	if provided == "" || expected == "" {
		return false
	}
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func handleSyncEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to send SYNC_EVENT", user.Username)
		return
	}

	var payload models.SyncPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid SYNC_EVENT payload:", err)
		return
	}

	if payload.CurrentTime < 0 {
		payload.CurrentTime = 0
	}
	if payload.CurrentTime > 86400 {
		payload.CurrentTime = 86400
	}
	if payload.Duration < 0 {
		payload.Duration = 0
	}

	isPlaying := payload.PlayerState == "PLAYING"
	UpdateRoomState(room.RoomID, payload.CurrentTime, isPlaying)

	payload.ServerAt = time.Now().UnixMilli()

	msg := models.Message{
		Action:  "SYNC_EVENT",
		Payload: mustMarshalRaw(payload),
	}

	BroadcastToRoom(room, mustMarshal(msg), user)
}

func handleChatEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !chatRateLimit.Allow(user.ID) {
		log.Printf("Chat rate limited for user %s", user.ID)
		return
	}

	var payload models.ChatPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid CHAT_EVENT payload:", err)
		return
	}

	text := strings.TrimSpace(payload.Text)
	if text == "" {
		return
	}
	if utf8.RuneCountInString(text) > maxChatTextLen {
		runes := []rune(text)
		text = string(runes[:maxChatTextLen])
	}

	safe := models.ChatPayload{
		RoomID:   room.RoomID,
		UserID:   user.ID,
		Username: user.Username,
		Text:     text,
		SentAt:   time.Now().UnixMilli(),
	}

	msg := models.Message{
		Action:  "CHAT_EVENT",
		Payload: mustMarshalRaw(safe),
	}

	BroadcastToRoom(room, mustMarshal(msg), nil)
}

func handleSetVideo(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to SET_VIDEO", user.Username)
		return
	}
	if !setVideoRateLimit.Allow(user.ID) {
		log.Printf("SET_VIDEO rate limited for user %s", user.ID)
		return
	}

	var payload models.SetVideoPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid SET_VIDEO payload:", err)
		return
	}

	if !isValidMediaURL(payload.URL) {
		log.Printf("Invalid SET_VIDEO URL from %s", user.Username)
		return
	}

	// Scrape off the readPump goroutine (IDLIX can take 30-90s).
	go processSetVideo(user, room, payload)
}

func processSetVideo(user *models.User, room *models.WatchRoom, payload models.SetVideoPayload) {
	var metadata *models.VideoMetadata

	if !isYouTubeURL(payload.URL) && !isDirectStreamURL(payload.URL) {
		broadcastScrapeStarted(room, payload.URL, "Resolving stream URL…")
		log.Printf("Scraping anime page: %s", payload.URL)
		scraped, err := ScrapeStreamURLCached(payload.URL)
		if err != nil {
			log.Printf("Scrape failed for %s: %v", payload.URL, err)
			errMsg := models.Message{
				Action: "SCRAPE_ERROR",
				Payload: mustMarshalRaw(models.ScrapeErrorPayload{
					OriginalURL: payload.URL,
					Error:       err.Error(),
				}),
			}
			// Details to requester; FINISHED to whole room so guests clear scrapeLoading.
			SendToUser(user, mustMarshal(errMsg))
			broadcastScrapeFinished(room, payload.URL)
			return
		}
		if scraped == nil || !isPlayableStreamURL(scraped.VideoURL) {
			vu := ""
			if scraped != nil {
				vu = scraped.VideoURL
			}
			log.Printf("Scrape produced non-playable URL for %s: %s", payload.URL, vu)
			InvalidateScrapeCache(payload.URL)
			errMsg := models.Message{
				Action: "SCRAPE_ERROR",
				Payload: mustMarshalRaw(models.ScrapeErrorPayload{
					OriginalURL: payload.URL,
					Error:       "Could not resolve a playable stream URL from this page",
				}),
			}
			SendToUser(user, mustMarshal(errMsg))
			// Always pair STARTED with FINISHED so room-wide loading UI clears.
			broadcastScrapeFinished(room, payload.URL)
			return
		}
		log.Printf("Scrape succeeded: %s - %s → %s", scraped.Title, scraped.Episode, truncateURL(scraped.VideoURL, 100))
		metadata = scraped
		broadcastScrapeFinished(room, payload.URL)
	} else {
		metadata = &models.VideoMetadata{
			VideoURL: payload.URL,
			Source:   extractDomainFromURL(payload.URL),
		}
	}

	SetRoomMetadata(room.RoomID, metadata)

	broadcastPayload := map[string]interface{}{
		"roomId":   room.RoomID,
		"videoUrl": metadata.VideoURL,
		"metadata": metadata,
	}
	msg := models.Message{
		Action:  "SET_VIDEO",
		Payload: mustMarshalRaw(broadcastPayload),
	}
	BroadcastToRoom(room, mustMarshal(msg), nil)
}

func handleAddToQueue(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !queueRateLimit.Allow(user.ID) {
		log.Printf("ADD_TO_QUEUE rate limited for user %s", user.ID)
		return
	}

	var payload models.AddToQueuePayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid ADD_TO_QUEUE payload:", err)
		return
	}

	if !isValidMediaURL(payload.URL) {
		log.Printf("Invalid ADD_TO_QUEUE URL from %s", user.Username)
		return
	}

	room.Mutex.RLock()
	qLen := len(room.Queue)
	room.Mutex.RUnlock()
	if qLen >= maxQueueSize {
		log.Printf("Queue full in room %s", room.RoomID)
		return
	}

	// Scrape off the readPump goroutine.
	go processAddToQueue(user, room, payload.URL)
}

func processAddToQueue(user *models.User, room *models.WatchRoom, pageURL string) {
	room.Mutex.RLock()
	qLen := len(room.Queue)
	room.Mutex.RUnlock()
	if qLen >= maxQueueSize {
		return
	}

	needsScrape := !isYouTubeURL(pageURL) && !isDirectStreamURL(pageURL)
	if needsScrape {
		broadcastScrapeStarted(room, pageURL, "Resolving stream URL…")
	}

	queueItem, err := AddToQueue(room.RoomID, pageURL)
	if err != nil {
		log.Printf("Failed to add to queue: %v", err)
		errMsg := models.Message{
			Action: "SCRAPE_ERROR",
			Payload: mustMarshalRaw(models.ScrapeErrorPayload{
				OriginalURL: pageURL,
				Error:       err.Error(),
			}),
		}
		SendToUser(user, mustMarshal(errMsg))
		// Pair STARTED with FINISHED on failure so guests do not stick on loading.
		if needsScrape {
			broadcastScrapeFinished(room, pageURL)
		}
		return
	}

	if needsScrape {
		broadcastScrapeFinished(room, pageURL)
	}

	queue := GetQueue(room.RoomID)
	broadcastMsg := models.Message{
		Action: "QUEUE_UPDATE",
		Payload: mustMarshalRaw(models.QueueUpdatePayload{
			RoomID: room.RoomID,
			Queue:  queue,
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), nil)

	log.Printf("User %s added to queue: %s", user.Username, queueItem.Title)
}

func handleRemoveFromQueue(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to remove from queue", user.Username)
		return
	}

	var payload models.RemoveFromQueuePayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid REMOVE_FROM_QUEUE payload:", err)
		return
	}

	if err := RemoveFromQueue(room.RoomID, payload.ItemID); err != nil {
		log.Printf("Failed to remove from queue: %v", err)
		return
	}

	queue := GetQueue(room.RoomID)
	broadcastMsg := models.Message{
		Action: "QUEUE_UPDATE",
		Payload: mustMarshalRaw(models.QueueUpdatePayload{
			RoomID: room.RoomID,
			Queue:  queue,
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), nil)
}

func handleSkipToNext(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to skip to next", user.Username)
		return
	}

	advanceAndBroadcast(user, room)
}

func handleClearQueue(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to clear queue", user.Username)
		return
	}

	if err := ClearQueue(room.RoomID); err != nil {
		log.Printf("Failed to clear queue: %v", err)
		return
	}

	broadcastMsg := models.Message{
		Action: "QUEUE_UPDATE",
		Payload: mustMarshalRaw(models.QueueUpdatePayload{
			RoomID: room.RoomID,
			Queue:  []*models.QueueItem{},
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), nil)
}

func handleEpisodeEnded(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to trigger EPISODE_ENDED", user.Username)
		return
	}

	now := time.Now().UnixNano()
	val, _ := lastEpisodeEnded.LoadOrStore(room.RoomID, &atomic.Int64{})
	lastPtr := val.(*atomic.Int64)
	last := lastPtr.Load()
	if now-last < int64(episodeEndedDebounce) {
		log.Printf("EPISODE_ENDED debounced for room %s (too soon)", room.RoomID)
		return
	}
	if !lastPtr.CompareAndSwap(last, now) {
		return
	}

	advanceAndBroadcast(user, room)
}

func advanceAndBroadcast(user *models.User, room *models.WatchRoom) {
	nextItem, err := AdvanceQueue(room.RoomID)
	if err != nil {
		log.Printf("Auto-advance failed: %v", err)
		return
	}

	videoChangedMsg := models.Message{
		Action: "CURRENT_VIDEO_CHANGED",
		Payload: mustMarshalRaw(map[string]interface{}{
			"roomId":   room.RoomID,
			"videoUrl": nextItem.URL,
			"metadata": nextItem,
		}),
	}
	BroadcastToRoom(room, mustMarshal(videoChangedMsg), nil)

	queue := GetQueue(room.RoomID)
	queueMsg := models.Message{
		Action: "QUEUE_UPDATE",
		Payload: mustMarshalRaw(models.QueueUpdatePayload{
			RoomID: room.RoomID,
			Queue:  queue,
		}),
	}
	BroadcastToRoom(room, mustMarshal(queueMsg), nil)

	log.Printf("Advanced to: %s - %s (by host %s)", nextItem.Title, nextItem.Episode, user.Username)
}

func handleTogglePublic(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to toggle public status", user.Username)
		return
	}

	var payload models.TogglePublicPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid TOGGLE_PUBLIC payload:", err)
		return
	}

	room.Mutex.Lock()
	room.IsPublic = payload.IsPublic
	room.Mutex.Unlock()

	status := "private"
	if payload.IsPublic {
		status = "public"
	}
	log.Printf("Room %s set to %s by host %s", room.RoomID, status, user.Username)

	broadcastMsg := models.Message{
		Action: "TOGGLE_PUBLIC",
		Payload: mustMarshalRaw(map[string]interface{}{
			"isPublic": payload.IsPublic,
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), nil)
}

func handleTransferHost(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to TRANSFER_HOST", user.Username)
		return
	}

	var payload models.TransferHostPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid TRANSFER_HOST payload:", err)
		return
	}

	targetID := strings.TrimSpace(payload.TargetUserID)
	if targetID == "" || targetID == user.ID {
		return
	}

	room.Mutex.Lock()
	target, ok := room.Clients[targetID]
	if !ok {
		room.Mutex.Unlock()
		log.Printf("TRANSFER_HOST: target %s not in room %s", targetID, room.RoomID)
		return
	}
	if room.HostID != user.ID {
		room.Mutex.Unlock()
		log.Printf("TRANSFER_HOST: %s is no longer host of %s", user.Username, room.RoomID)
		return
	}

	if oldHost, exists := room.Clients[room.HostID]; exists {
		oldHost.IsHost = false
	}
	user.IsHost = false
	target.IsHost = true
	room.HostID = target.ID
	newHostID := target.ID
	newHostUsername := target.Username
	room.Mutex.Unlock()

	log.Printf("Host transferred in room %s: %s -> %s (%s)", room.RoomID, user.Username, newHostUsername, newHostID)

	broadcastMsg := models.Message{
		Action: "HOST_CHANGED",
		Payload: mustMarshalRaw(models.HostChangedPayload{
			RoomID:    room.RoomID,
			NewHostID: newHostID,
			Username:  newHostUsername,
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), nil)
}

func handleKickUser(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to KICK_USER", user.Username)
		return
	}

	var payload models.KickUserPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid KICK_USER payload:", err)
		return
	}

	targetID := strings.TrimSpace(payload.TargetUserID)
	if targetID == "" || targetID == user.ID {
		log.Printf("KICK_USER: host cannot kick self")
		return
	}

	room.Mutex.RLock()
	target, ok := room.Clients[targetID]
	isHostTarget := ok && room.HostID == targetID
	room.Mutex.RUnlock()
	if !ok {
		log.Printf("KICK_USER: target %s not in room %s", targetID, room.RoomID)
		return
	}
	if isHostTarget {
		log.Printf("KICK_USER: refuse kicking current host without transfer")
		return
	}

	targetUsername := target.Username
	roomID := room.RoomID
	conn := target.Conn

	kickedMsg := models.Message{
		Action: "KICKED",
		Payload: mustMarshalRaw(models.KickedPayload{
			RoomID: roomID,
			Reason: "kicked_by_host",
		}),
	}
	// Enqueue via writePump only (never WriteMessage concurrently — not thread-safe).
	SendToUser(target, mustMarshal(kickedMsg))

	// Prevent readPump defer from re-broadcasting USER_LEFT after we close the conn.
	target.Username = ""
	RemoveUserFromRoom(roomID, targetID)

	leftMsg := models.Message{
		Action: "USER_LEFT",
		Payload: mustMarshalRaw(models.UserLeftPayload{
			UserID:   targetID,
			Username: targetUsername,
		}),
	}
	if r, ok := GetRoom(roomID); ok {
		BroadcastToRoom(r, mustMarshal(leftMsg), nil)
	}

	// Allow writePump to flush KICKED before TCP close so client can set permanentClose.
	// Without this delay, onclose races ahead of onmessage → auto-reconnect.
	// Only Close the conn from this goroutine — never WriteControl/WriteMessage
	// concurrently with writePump (gorilla websocket is not thread-safe for writes).
	// Conn.Close alone is safe and causes readPump/writePump to exit.
	go func() {
		time.Sleep(400 * time.Millisecond)
		if conn != nil {
			_ = conn.Close()
		}
	}()

	log.Printf("User %s (%s) kicked from room %s by host %s", targetUsername, targetID, roomID, user.Username)
}

func handleSetRoomName(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !user.IsHost {
		log.Printf("Non-host %s tried to SET_ROOM_NAME", user.Username)
		return
	}

	var payload models.SetRoomNamePayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid SET_ROOM_NAME payload:", err)
		return
	}

	name, ok := sanitizeRoomName(payload.RoomName)
	if !ok {
		log.Printf("SET_ROOM_NAME: invalid name from %s", user.Username)
		return
	}

	room.Mutex.Lock()
	room.RoomName = name
	room.Mutex.Unlock()

	log.Printf("Room %s renamed to %q by host %s", room.RoomID, name, user.Username)

	broadcastMsg := models.Message{
		Action: "ROOM_NAME_CHANGED",
		Payload: mustMarshalRaw(models.RoomNameChangedPayload{
			RoomID:   room.RoomID,
			RoomName: name,
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), nil)
}

func handleReaction(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !reactionRateLimit.Allow(user.ID) {
		return
	}

	var payload models.ReactionPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid REACTION payload:", err)
		return
	}

	emoji := strings.TrimSpace(payload.Emoji)
	if !isValidReaction(emoji) {
		return
	}

	broadcastMsg := models.Message{
		Action: "REACTION",
		Payload: mustMarshalRaw(models.ReactionBroadcastPayload{
			RoomID:   room.RoomID,
			UserID:   user.ID,
			Username: user.Username,
			Emoji:    emoji,
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), nil)
}

func handleTyping(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	if !typingRateLimit.Allow(user.ID) {
		return
	}

	var payload models.TypingPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid TYPING payload:", err)
		return
	}

	broadcastMsg := models.Message{
		Action: "TYPING",
		Payload: mustMarshalRaw(models.TypingBroadcastPayload{
			UserID:   user.ID,
			Username: user.Username,
		}),
	}
	BroadcastToRoom(room, mustMarshal(broadcastMsg), user)
}

func sanitizeRoomName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", true
	}
	if utf8.RuneCountInString(name) > maxRoomNameLen {
		runes := []rune(name)
		name = string(runes[:maxRoomNameLen])
	}
	// Reject control chars and HTML-ish brackets; allow unicode letters/numbers.
	for _, r := range name {
		if r < 0x20 || r == 0x7f || r == '<' || r == '>' {
			return "", false
		}
	}
	return name, true
}

func isValidReaction(emoji string) bool {
	if emoji == "" {
		return false
	}
	if utf8.RuneCountInString(emoji) > maxEmojiLen {
		return false
	}
	if _, ok := allowedReactions[emoji]; ok {
		return true
	}
	// Allow single emoji-like grapheme (no control/space chars).
	for _, r := range emoji {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return utf8.RuneCountInString(emoji) <= 4
}

func broadcastScrapeStarted(room *models.WatchRoom, originalURL, message string) {
	msg := models.Message{
		Action: "SCRAPE_STARTED",
		Payload: mustMarshalRaw(models.ScrapeStartedPayload{
			OriginalURL: originalURL,
			Message:     message,
		}),
	}
	BroadcastToRoom(room, mustMarshal(msg), nil)
}

func broadcastScrapeFinished(room *models.WatchRoom, originalURL string) {
	msg := models.Message{
		Action: "SCRAPE_FINISHED",
		Payload: mustMarshalRaw(map[string]interface{}{
			"originalUrl": originalURL,
		}),
	}
	BroadcastToRoom(room, mustMarshal(msg), nil)
}

func extractDomainFromURL(rawURL string) string {
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func mustMarshalRaw(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("Marshal error:", err)
		return json.RawMessage("{}")
	}
	return data
}

func safeSend(ch chan []byte, message []byte, clientID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered send on closed channel for client %s: %v", clientID, r)
		}
	}()
	select {
	case ch <- message:
	default:
		log.Println("Client send buffer full:", clientID)
	}
}

func BroadcastToRoom(room *models.WatchRoom, message []byte, exclude *models.User) {
	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	for _, client := range room.Clients {
		if exclude != nil && client.ID == exclude.ID {
			continue
		}
		safeSend(client.Send, message, client.ID)
	}
}

func SendToUser(user *models.User, message []byte) {
	if user == nil {
		return
	}
	safeSend(user.Send, message, user.ID)
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Println("Marshal error:", err)
		return []byte("{}")
	}
	return data
}

// PruneEpisodeEnded removes debounce state for a deleted room.
func PruneEpisodeEnded(roomID string) {
	lastEpisodeEnded.Delete(roomID)
}
