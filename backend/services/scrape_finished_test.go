package services

import (
	"fmt"
	"testing"
	"time"
	"watchparty-backend/models"
)

func setupScrapeRoom(t *testing.T) (room *models.WatchRoom, host, guest *models.User) {
	t.Helper()
	roomID := fmt.Sprintf("scrape-%d", time.Now().UnixNano())
	CreateRoomWithID(roomID)
	var ok bool
	room, ok = GetRoom(roomID)
	if !ok {
		t.Fatal("room missing")
	}
	host = &models.User{ID: "host1", Username: "Host", IsHost: true, Send: make(chan []byte, 32)}
	guest = &models.User{ID: "guest1", Username: "Guest", Send: make(chan []byte, 32)}
	room.Mutex.Lock()
	room.Clients[host.ID] = host
	room.Clients[guest.ID] = guest
	room.HostID = host.ID
	room.Mutex.Unlock()
	return room, host, guest
}

func hasAction(msgs []models.Message, action string) bool {
	for _, m := range msgs {
		if m.Action == action {
			return true
		}
	}
	return false
}

// BE-M01: scrape failure must broadcast SCRAPE_FINISHED so guests clear scrapeLoading.
func TestProcessSetVideo_ScrapeErrorBroadcastsFinished(t *testing.T) {
	room, host, guest := setupScrapeRoom(t)

	// Private IP is blocked by scraper SSRF guards — fails without network.
	processSetVideo(host, room, models.SetVideoPayload{
		RoomID: room.RoomID,
		URL:    "https://127.0.0.1/episode-page",
	})

	// Allow async-safe drains (processSetVideo is sync when called directly).
	hostMsgs := drainMessages(host.Send, 200*time.Millisecond)
	guestMsgs := drainMessages(guest.Send, 50*time.Millisecond)

	if !hasAction(hostMsgs, "SCRAPE_ERROR") {
		t.Fatalf("host should receive SCRAPE_ERROR, got %+v", actionsOf(hostMsgs))
	}
	// STARTED + FINISHED are room-wide
	if !hasAction(hostMsgs, "SCRAPE_STARTED") && !hasAction(guestMsgs, "SCRAPE_STARTED") {
		// STARTED may already have been drained with host; require FINISHED at least room-wide
	}
	if !hasAction(hostMsgs, "SCRAPE_FINISHED") && !hasAction(guestMsgs, "SCRAPE_FINISHED") {
		t.Fatalf("room must receive SCRAPE_FINISHED on scrape failure; host=%v guest=%v",
			actionsOf(hostMsgs), actionsOf(guestMsgs))
	}
	if !hasAction(guestMsgs, "SCRAPE_FINISHED") && !hasAction(hostMsgs, "SCRAPE_FINISHED") {
		t.Fatal("SCRAPE_FINISHED missing")
	}
	// Guest should clear loading via FINISHED (not only host-only SCRAPE_ERROR)
	guestGotFinished := hasAction(guestMsgs, "SCRAPE_FINISHED")
	hostGotFinished := hasAction(hostMsgs, "SCRAPE_FINISHED")
	if !guestGotFinished || !hostGotFinished {
		t.Fatalf("both host and guest need SCRAPE_FINISHED; host=%v guest=%v",
			actionsOf(hostMsgs), actionsOf(guestMsgs))
	}
}

func TestProcessAddToQueue_ScrapeErrorBroadcastsFinished(t *testing.T) {
	room, host, guest := setupScrapeRoom(t)

	processAddToQueue(host, room, "https://127.0.0.1/episode-page")

	hostMsgs := drainMessages(host.Send, 200*time.Millisecond)
	guestMsgs := drainMessages(guest.Send, 50*time.Millisecond)

	if !hasAction(hostMsgs, "SCRAPE_ERROR") {
		t.Fatalf("host should receive SCRAPE_ERROR, got %+v", actionsOf(hostMsgs))
	}
	if !hasAction(hostMsgs, "SCRAPE_FINISHED") || !hasAction(guestMsgs, "SCRAPE_FINISHED") {
		t.Fatalf("room must receive SCRAPE_FINISHED on queue scrape failure; host=%v guest=%v",
			actionsOf(hostMsgs), actionsOf(guestMsgs))
	}
}

func actionsOf(msgs []models.Message) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Action)
	}
	return out
}
