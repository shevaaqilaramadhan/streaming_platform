package services

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"watchparty-backend/models"
)

func TestHandleJoinEvent_IdempotentSameUser(t *testing.T) {
	roomID := fmt.Sprintf("join-idem-%d", time.Now().UnixNano())
	CreateRoomWithID(roomID)
	room, ok := GetRoom(roomID)
	if !ok {
		t.Fatal("room missing")
	}

	send := make(chan []byte, 16)
	user := &models.User{
		ID:   "u1",
		Send: send,
	}

	payload, _ := json.Marshal(models.JoinPayload{RoomID: roomID, Username: "Alice"})
	handleJoinEvent(user, room, payload)
	handleJoinEvent(user, room, payload) // double JOIN same connection

	room.Mutex.RLock()
	n := len(room.Clients)
	host := room.HostID
	room.Mutex.RUnlock()

	if n != 1 {
		t.Fatalf("clients = %d, want 1 (idempotent re-JOIN)", n)
	}
	if host != "u1" {
		t.Fatalf("host = %q, want u1", host)
	}
	if !user.IsHost {
		t.Fatal("user should be host")
	}

	// At least one ROOM_INIT; no panic
	deadline := time.After(200 * time.Millisecond)
	gotInit := 0
	for {
		select {
		case msg := <-send:
			var m models.Message
			if json.Unmarshal(msg, &m) == nil && m.Action == "ROOM_INIT" {
				gotInit++
			}
		case <-deadline:
			if gotInit < 1 {
				t.Fatal("expected at least one ROOM_INIT")
			}
			return
		}
	}
}

func TestHandleJoinEvent_RoomFull(t *testing.T) {
	roomID := fmt.Sprintf("join-full-%d", time.Now().UnixNano())
	CreateRoomWithID(roomID)
	room, _ := GetRoom(roomID)

	room.Mutex.Lock()
	for i := 0; i < MaxUsersPerRoom; i++ {
		uid := fmt.Sprintf("full-%d", i)
		room.Clients[uid] = &models.User{ID: uid, Username: "x", Send: make(chan []byte, 1)}
	}
	room.Mutex.Unlock()

	send := make(chan []byte, 4)
	user := &models.User{ID: "newcomer", Send: send}
	payload, _ := json.Marshal(models.JoinPayload{RoomID: roomID, Username: "Bob"})
	handleJoinEvent(user, room, payload)

	room.Mutex.RLock()
	_, in := room.Clients["newcomer"]
	n := len(room.Clients)
	room.Mutex.RUnlock()
	if in {
		t.Fatal("newcomer should not be admitted when full")
	}
	if n != MaxUsersPerRoom {
		t.Fatalf("clients = %d, want %d", n, MaxUsersPerRoom)
	}
}

func TestRemoveUserFromRoom_IdempotentAndHostReassign(t *testing.T) {
	roomID := fmt.Sprintf("rm-host-%d", time.Now().UnixNano())
	CreateRoomWithID(roomID)
	room, _ := GetRoom(roomID)

	hSend := make(chan []byte, 8)
	gSend := make(chan []byte, 8)
	host := &models.User{ID: "host1", Username: "Host", IsHost: true, Send: hSend}
	guest := &models.User{ID: "guest1", Username: "Guest", Send: gSend}

	room.Mutex.Lock()
	room.Clients[host.ID] = host
	room.Clients[guest.ID] = guest
	room.HostID = host.ID
	room.Mutex.Unlock()

	RemoveUserFromRoom(roomID, host.ID)
	// second remove is no-op / safe
	RemoveUserFromRoom(roomID, host.ID)

	room.Mutex.RLock()
	_, hostIn := room.Clients[host.ID]
	newHost := room.HostID
	guestHost := guest.IsHost
	n := len(room.Clients)
	room.Mutex.RUnlock()

	if hostIn {
		t.Fatal("host should be removed")
	}
	if n != 1 {
		t.Fatalf("clients = %d, want 1", n)
	}
	if newHost != guest.ID {
		t.Fatalf("new host = %q, want guest1", newHost)
	}
	if !guestHost {
		t.Fatal("guest should be promoted to host")
	}
}

func TestHandleJoinEvent_RejoinAfterLeave(t *testing.T) {
	// Simulates reconnect: old user leaves, new user ID joins same room.
	// Legacy room (no token): first joiner after empty becomes host.
	roomID := fmt.Sprintf("rejoin-%d", time.Now().UnixNano())
	CreateRoomWithID(roomID)
	room, _ := GetRoom(roomID)

	oldSend := make(chan []byte, 8)
	oldUser := &models.User{ID: "old-id", Send: oldSend}
	payload, _ := json.Marshal(models.JoinPayload{RoomID: roomID, Username: "Alice"})
	handleJoinEvent(oldUser, room, payload)

	RemoveUserFromRoom(roomID, oldUser.ID)

	newSend := make(chan []byte, 8)
	newUser := &models.User{ID: "new-id", Send: newSend}
	handleJoinEvent(newUser, room, payload)

	room.Mutex.RLock()
	_, oldIn := room.Clients["old-id"]
	_, newIn := room.Clients["new-id"]
	host := room.HostID
	n := len(room.Clients)
	room.Mutex.RUnlock()

	if oldIn {
		t.Fatal("old connection should be gone")
	}
	if !newIn {
		t.Fatal("new connection should be in room")
	}
	if n != 1 {
		t.Fatalf("clients = %d, want 1", n)
	}
	if host != "new-id" {
		t.Fatalf("host = %q, want new-id (reassigned after empty)", host)
	}
	if !newUser.IsHost {
		t.Fatal("rejoined sole user should be host")
	}
}

func TestHandleJoinEvent_HostTokenSingleUse(t *testing.T) {
	roomID, hostToken := CreateRoom()
	room, ok := GetRoom(roomID)
	if !ok {
		t.Fatal("room missing")
	}
	if hostToken == "" {
		t.Fatal("expected non-empty hostToken")
	}

	// Guest without token must not become host while token is pending
	guestSend := make(chan []byte, 8)
	guest := &models.User{ID: "guest1", Send: guestSend}
	guestPayload, _ := json.Marshal(models.JoinPayload{RoomID: roomID, Username: "Guest"})
	handleJoinEvent(guest, room, guestPayload)
	if guest.IsHost {
		t.Fatal("guest without token must not be host")
	}
	room.Mutex.RLock()
	hostID := room.HostID
	tokenStill := room.HostToken
	room.Mutex.RUnlock()
	if hostID != "" {
		t.Fatalf("host should be empty before claim, got %q", hostID)
	}
	if tokenStill == "" {
		t.Fatal("token should remain until claimed")
	}

	// Wrong token
	badSend := make(chan []byte, 8)
	bad := &models.User{ID: "bad1", Send: badSend}
	badPayload, _ := json.Marshal(models.JoinPayload{RoomID: roomID, Username: "Bad", HostToken: "wrong-token-value-xxxxxxxx"})
	handleJoinEvent(bad, room, badPayload)
	if bad.IsHost {
		t.Fatal("wrong token must not grant host")
	}

	// Valid token → host; token consumed
	creatorSend := make(chan []byte, 8)
	creator := &models.User{ID: "creator1", Send: creatorSend}
	okPayload, _ := json.Marshal(models.JoinPayload{RoomID: roomID, Username: "Creator", HostToken: hostToken})
	handleJoinEvent(creator, room, okPayload)
	if !creator.IsHost {
		t.Fatal("valid token should grant host")
	}
	room.Mutex.RLock()
	if room.HostID != creator.ID {
		t.Fatalf("host = %q, want creator1", room.HostID)
	}
	if room.HostToken != "" {
		t.Fatal("host token must be cleared after single-use claim")
	}
	room.Mutex.RUnlock()

	// Replay token after claim → no privilege
	replaySend := make(chan []byte, 8)
	replay := &models.User{ID: "replay1", Send: replaySend}
	replayPayload, _ := json.Marshal(models.JoinPayload{RoomID: roomID, Username: "Replay", HostToken: hostToken})
	handleJoinEvent(replay, room, replayPayload)
	if replay.IsHost {
		t.Fatal("replayed single-use token must not grant host")
	}
	room.Mutex.RLock()
	if room.HostID != creator.ID {
		t.Fatalf("host should stay creator1, got %q", room.HostID)
	}
	room.Mutex.RUnlock()
}
