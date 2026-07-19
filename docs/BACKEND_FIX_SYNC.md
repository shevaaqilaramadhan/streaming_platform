# Backend Fix Required — User Synchronization and State Sharing

## Problem

Guests are stuck seeing **"Waiting for the host to start a video..."** because the backend never registers connected users into the room's active clients list.

Specifically:
1. When a WebSocket connection is upgraded, the backend creates a `User` struct but **never adds it to `room.Clients`**.
2. Because `room.Clients` is empty, any call to `BroadcastToRoom()` (for `SET_VIDEO`, `SYNC_EVENT`, etc.) loops over an empty map and **broadcasts to no one**.
3. When `handleJoinEvent` is processed, `GetRoomState()` is called, but since `room.Clients` is empty, it returns an empty participants list, and does not register the join.
4. When any client disconnects, `RemoveUserFromRoom()` checks `len(room.Clients)`. Since it is `0`, the server **deletes the room from memory entirely**!

---

## Fix — `backend/services/message_handler.go`

In `backend/services/message_handler.go`, update `handleJoinEvent` to register the connecting user into `room.Clients` (with appropriate mutex locking) once we receive their `JOIN_EVENT`.

### Before
```go
func handleJoinEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	var payload models.JoinPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid JOIN_EVENT payload:", err)
		return
	}

	user.Username = payload.Username
	user.IsHost = payload.IsHost

	currentVideo, currentTime, isPlaying, participants := GetRoomState(room.RoomID)
```

### After
```go
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

	// Fetch current state (which now correctly includes the new user)
	currentVideo, currentTime, isPlaying, participants := GetRoomState(room.RoomID)
```

---

## Steps to Verify

1. Apply this fix to `backend/services/message_handler.go`.
2. Re-compile and restart the Go backend:
   ```bash
   fuser -k 8080/tcp # Kill existing process on 8080
   go run main.go
   ```
3. Open the app as host, load a video, and play it.
4. Open the room link in an Incognito window as guest, choose a username, and join. The guest should now immediately seek to the host's current time, play in sync, and both clients should appear in the participant list!
