package services

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"watchparty-backend/models"
)

func setupHostGuestRoom(t *testing.T) (roomID string, room *models.WatchRoom, host, guest *models.User) {
	t.Helper()
	roomID = fmt.Sprintf("mod-%d", time.Now().UnixNano())
	CreateRoomWithID(roomID)
	var ok bool
	room, ok = GetRoom(roomID)
	if !ok {
		t.Fatal("room missing")
	}
	host = &models.User{ID: "host1", Username: "Host", IsHost: true, Send: make(chan []byte, 16)}
	guest = &models.User{ID: "guest1", Username: "Guest", Send: make(chan []byte, 16)}
	room.Mutex.Lock()
	room.Clients[host.ID] = host
	room.Clients[guest.ID] = guest
	room.HostID = host.ID
	room.Mutex.Unlock()
	return roomID, room, host, guest
}

func drainMessages(ch <-chan []byte, timeout time.Duration) []models.Message {
	var out []models.Message
	deadline := time.After(timeout)
	for {
		select {
		case raw := <-ch:
			var m models.Message
			if json.Unmarshal(raw, &m) == nil {
				out = append(out, m)
			}
		case <-deadline:
			return out
		}
	}
}

func TestTransferHost(t *testing.T) {
	_, room, host, guest := setupHostGuestRoom(t)

	payload, _ := json.Marshal(models.TransferHostPayload{
		RoomID:       room.RoomID,
		TargetUserID: guest.ID,
	})
	handleTransferHost(host, room, payload)

	room.Mutex.RLock()
	hostID := room.HostID
	hostIsHost := host.IsHost
	guestIsHost := guest.IsHost
	room.Mutex.RUnlock()

	if hostID != guest.ID {
		t.Fatalf("HostID = %q, want guest1", hostID)
	}
	if hostIsHost {
		t.Fatal("old host should not be IsHost")
	}
	if !guestIsHost {
		t.Fatal("target should be IsHost")
	}

	msgs := drainMessages(host.Send, 100*time.Millisecond)
	got := false
	for _, m := range msgs {
		if m.Action == "HOST_CHANGED" {
			var p models.HostChangedPayload
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatal(err)
			}
			if p.NewHostID != guest.ID || p.Username != "Guest" {
				t.Fatalf("HOST_CHANGED payload = %+v", p)
			}
			got = true
		}
	}
	if !got {
		t.Fatal("expected HOST_CHANGED broadcast")
	}
}

func TestTransferHost_NonHostRejected(t *testing.T) {
	_, room, host, guest := setupHostGuestRoom(t)
	payload, _ := json.Marshal(models.TransferHostPayload{
		RoomID:       room.RoomID,
		TargetUserID: host.ID,
	})
	handleTransferHost(guest, room, payload)

	room.Mutex.RLock()
	hostID := room.HostID
	room.Mutex.RUnlock()
	if hostID != host.ID {
		t.Fatalf("host should remain host1, got %q", hostID)
	}
}

func TestTransferHost_TargetNotInRoom(t *testing.T) {
	_, room, host, _ := setupHostGuestRoom(t)
	payload, _ := json.Marshal(models.TransferHostPayload{
		RoomID:       room.RoomID,
		TargetUserID: "missing",
	})
	handleTransferHost(host, room, payload)

	room.Mutex.RLock()
	hostID := room.HostID
	room.Mutex.RUnlock()
	if hostID != host.ID {
		t.Fatalf("host should remain host1, got %q", hostID)
	}
}

func TestKickUser(t *testing.T) {
	_, room, host, guest := setupHostGuestRoom(t)

	payload, _ := json.Marshal(models.KickUserPayload{
		RoomID:       room.RoomID,
		TargetUserID: guest.ID,
	})
	handleKickUser(host, room, payload)

	room.Mutex.RLock()
	_, stillIn := room.Clients[guest.ID]
	n := len(room.Clients)
	room.Mutex.RUnlock()
	if stillIn {
		t.Fatal("guest should be removed")
	}
	if n != 1 {
		t.Fatalf("clients = %d, want 1", n)
	}

	// Target should have received KICKED
	gotKicked := false
	deadline := time.After(100 * time.Millisecond)
	for !gotKicked {
		select {
		case raw := <-guest.Send:
			var m models.Message
			if json.Unmarshal(raw, &m) == nil && m.Action == "KICKED" {
				gotKicked = true
			}
		case <-deadline:
			t.Fatal("expected KICKED on target")
		}
	}

	// Host should see USER_LEFT
	msgs := drainMessages(host.Send, 100*time.Millisecond)
	gotLeft := false
	for _, m := range msgs {
		if m.Action == "USER_LEFT" {
			var p models.UserLeftPayload
			_ = json.Unmarshal(m.Payload, &p)
			if p.UserID == guest.ID {
				gotLeft = true
			}
		}
	}
	if !gotLeft {
		t.Fatal("expected USER_LEFT broadcast")
	}
}

func TestKickUser_CannotKickSelf(t *testing.T) {
	_, room, host, _ := setupHostGuestRoom(t)
	payload, _ := json.Marshal(models.KickUserPayload{
		RoomID:       room.RoomID,
		TargetUserID: host.ID,
	})
	handleKickUser(host, room, payload)

	room.Mutex.RLock()
	_, stillIn := room.Clients[host.ID]
	room.Mutex.RUnlock()
	if !stillIn {
		t.Fatal("host should not kick self")
	}
}

func TestSetRoomName(t *testing.T) {
	_, room, host, guest := setupHostGuestRoom(t)

	payload, _ := json.Marshal(models.SetRoomNamePayload{
		RoomID:   room.RoomID,
		RoomName: "  Anime Night  ",
	})
	handleSetRoomName(host, room, payload)

	room.Mutex.RLock()
	name := room.RoomName
	room.Mutex.RUnlock()
	if name != "Anime Night" {
		t.Fatalf("RoomName = %q, want Anime Night", name)
	}

	msgs := drainMessages(guest.Send, 100*time.Millisecond)
	got := false
	for _, m := range msgs {
		if m.Action == "ROOM_NAME_CHANGED" {
			var p models.RoomNameChangedPayload
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatal(err)
			}
			if p.RoomName != "Anime Night" || p.RoomID != room.RoomID {
				t.Fatalf("payload = %+v", p)
			}
			got = true
		}
	}
	if !got {
		t.Fatal("expected ROOM_NAME_CHANGED")
	}
}

func TestSetRoomName_InvalidCharset(t *testing.T) {
	_, room, host, _ := setupHostGuestRoom(t)
	payload, _ := json.Marshal(models.SetRoomNamePayload{
		RoomID:   room.RoomID,
		RoomName: "<script>alert(1)</script>",
	})
	handleSetRoomName(host, room, payload)

	room.Mutex.RLock()
	name := room.RoomName
	room.Mutex.RUnlock()
	if name != "" {
		t.Fatalf("invalid name should be rejected, got %q", name)
	}
}

func TestSetRoomName_NonHostRejected(t *testing.T) {
	_, room, _, guest := setupHostGuestRoom(t)
	payload, _ := json.Marshal(models.SetRoomNamePayload{
		RoomID:   room.RoomID,
		RoomName: "Hacked",
	})
	handleSetRoomName(guest, room, payload)

	room.Mutex.RLock()
	name := room.RoomName
	room.Mutex.RUnlock()
	if name != "" {
		t.Fatalf("non-host must not rename, got %q", name)
	}
}

func TestReactionAndTyping(t *testing.T) {
	_, room, host, guest := setupHostGuestRoom(t)

	rPayload, _ := json.Marshal(models.ReactionPayload{RoomID: room.RoomID, Emoji: "🔥"})
	handleReaction(guest, room, rPayload)

	tPayload, _ := json.Marshal(models.TypingPayload{RoomID: room.RoomID})
	handleTyping(guest, room, tPayload)

	hostMsgs := drainMessages(host.Send, 100*time.Millisecond)
	gotReaction, gotTyping := false, false
	for _, m := range hostMsgs {
		switch m.Action {
		case "REACTION":
			var p models.ReactionBroadcastPayload
			_ = json.Unmarshal(m.Payload, &p)
			if p.Emoji == "🔥" && p.UserID == guest.ID {
				gotReaction = true
			}
		case "TYPING":
			var p models.TypingBroadcastPayload
			_ = json.Unmarshal(m.Payload, &p)
			if p.UserID == guest.ID && p.Username == "Guest" {
				gotTyping = true
			}
		}
	}
	if !gotReaction {
		t.Fatal("expected REACTION on host")
	}
	if !gotTyping {
		t.Fatal("expected TYPING on host (exclude sender)")
	}

	// Typing should not echo to sender
	guestMsgs := drainMessages(guest.Send, 50*time.Millisecond)
	for _, m := range guestMsgs {
		if m.Action == "TYPING" {
			t.Fatal("TYPING must exclude sender")
		}
	}
}

func TestSanitizeRoomName(t *testing.T) {
	if name, ok := sanitizeRoomName("  ok name  "); !ok || name != "ok name" {
		t.Fatalf("got %q ok=%v", name, ok)
	}
	if _, ok := sanitizeRoomName("bad<script>"); ok {
		t.Fatal("should reject angle brackets")
	}
	runes := make([]rune, 100)
	for i := range runes {
		runes[i] = 'a'
	}
	name, ok := sanitizeRoomName(string(runes))
	if !ok {
		t.Fatal("long ascii should truncate not reject")
	}
	if len([]rune(name)) != maxRoomNameLen {
		t.Fatalf("len = %d, want %d", len([]rune(name)), maxRoomNameLen)
	}
}

func TestIsValidReaction(t *testing.T) {
	if !isValidReaction("🔥") {
		t.Fatal("fire emoji ok")
	}
	if !isValidReaction("gg") {
		t.Fatal("gg allowlist ok")
	}
	if isValidReaction("") {
		t.Fatal("empty invalid")
	}
	if isValidReaction("toolongemoji!!") {
		t.Fatal("too long invalid")
	}
}
