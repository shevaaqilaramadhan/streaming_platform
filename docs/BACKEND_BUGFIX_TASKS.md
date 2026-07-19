# Backend Bug Fix Tasks — WatchParty

> **Untuk backend agent**: Dokumen ini berisi task perbaikan bug yang sudah di-diagnosa dari analisis kode terbaru. Setiap task mandiri — kerjakan satu per satu sesuai urutan prioritas. Verifikasi setiap task sebelum lanjut ke task berikutnya.
>
> **Urutan prioritas yang diminta:** C04 → C03 → C07 → C01 → C06 → C02 → C08

---

## Task 1 — Fix SSRF Proxy (BE-C04) [KRITIS]

### Masalah
`handlers/proxy.go:47-90` — `/api/proxy?url=` bisa fetch **semua** URL HTTP/HTTPS tanpa batasan. Attacker bisa:
- Hit internal services: `http://127.0.0.1:8080/api/rooms`
- Akses Docker network: `http://host.docker.internal:*`
- Fetch cloud metadata: `http://169.254.169.254/latest/meta-data/`
- Scan internal network: `http://192.168.x.x:*`

### File yang Diubah
- `backend/handlers/proxy.go`

### Implementasi

Tambahkan fungsi validasi URL **sebelum** request dibuat (setelah line 59, sebelum line 62):

```go
import (
	"net"
	// ... existing imports
)

// isPrivateIP checks if an IP is in a private/reserved range (RFC1918, loopback, link-local, cloud metadata).
func isPrivateIP(host string) bool {
	// Resolve hostname to IP
	ips, err := net.LookupIP(host)
	if err != nil {
		// If we can't resolve, try parsing directly as IP
		ip := net.ParseIP(host)
		if ip == nil {
			return false // can't determine, allow (will fail at connection time)
		}
		ips = []net.IP{ip}
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
		// Block cloud metadata endpoint
		if ip.Equal(net.ParseIP("169.254.169.254")) || ip.Equal(net.ParseIP("fd00::ec2::254")) {
			return true
		}
		// Block Docker bridge networks
		if ip.Equal(net.ParseIP("host.docker.internal")) {
			return true
		}
	}
	return false
}

// validateProxyURL checks if a URL is safe to proxy (not pointing to internal services).
func validateProxyURL(targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("missing hostname")
	}

	// Block localhost variants
	blockedHosts := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "host.docker.internal"}
	for _, blocked := range blockedHosts {
		if host == blocked {
			return fmt.Errorf("access to %s is blocked", host)
		}
	}

	// Block private/reserved IP ranges
	if isPrivateIP(host) {
		return fmt.Errorf("access to private network (%s) is blocked", host)
	}

	return nil
}
```

Modifikasi `HandleStreamProxy` — tambah validasi **setelah** URL parsing (setelah line 59):

```go
	// SSRF protection — block internal/private network access
	if err := validateProxyURL(targetURL); err != nil {
		log.Printf("[Proxy] Blocked SSRF attempt: %s — %v", targetURL, err)
		http.Error(w, "Forbidden: "+err.Error(), http.StatusForbidden)
		return
	}
```

### Verifikasi
```bash
cd backend && go build ./...

# Test: harus BLOCKED (403)
curl "http://localhost:8080/api/proxy?url=http://127.0.0.1:8080/api/rooms"
curl "http://localhost:8080/api/proxy?url=http://169.254.169.254/latest/meta-data/"
curl "http://localhost:8080/api/proxy?url=http://localhost:8080"

# Test: harus ALLOWED (200 atau upstream error, bukan 403)
curl "http://localhost:8080/api/proxy?url=https://example.com/video.m3u8"
```

---

## Task 2 — Fix Host Privilege Escalation (BE-C03) [KRITIS]

### Masalah
`services/message_handler.go:70-71` — `user.IsHost = payload.IsHost` langsung trust dari client. Siapa pun bisa kirim `isHost: true` saat join dan dapat kontrol penuh (skip queue, remove video, toggle public).

`services/room_manager.go:129-154` — Saat host leave, `HostID` tidak pernah di-clear atau di-reassign. Room jadi tanpa host selamanya.

### File yang Diubah
- `backend/services/message_handler.go` — `handleJoinEvent`
- `backend/services/room_manager.go` — `RemoveUserFromRoom`

### Implementasi

#### 2a. Server-side host assignment di `handleJoinEvent`

Ubah `handleJoinEvent` (line 63-108). **Jangan** trust `payload.IsHost` dari client. Server yang menentukan host:

```go
func handleJoinEvent(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	var payload models.JoinPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		log.Println("Invalid JOIN_EVENT payload:", err)
		return
	}

	user.Username = payload.Username
	// JANGAN set user.IsHost dari payload — server yang menentukan

	room.Mutex.Lock()
	room.Clients[user.ID] = user

	// Server-side host assignment: first joiner = host
	if room.HostID == "" {
		room.HostID = user.ID
		user.IsHost = true
		log.Printf("User %s (%s) assigned as host for room %s", user.Username, user.ID, room.RoomID)
	} else {
		user.IsHost = (room.HostID == user.ID)
	}
	room.Mutex.Unlock()

	// ... rest of the function unchanged (GetRoomState, ROOM_INIT, JOIN_EVENT broadcast)
}
```

#### 2b. Host reassignment saat host leave di `RemoveUserFromRoom`

Modifikasi `RemoveUserFromRoom` (line 129-154). Saat host leave, promote client berikutnya:

```go
func RemoveUserFromRoom(roomID, userID string) {
	room, exists := GetRoom(roomID)
	if !exists {
		return
	}

	room.Mutex.Lock()
	delete(room.Clients, userID)

	// Host reassignment: if the leaving user was host, promote next client
	if room.HostID == userID {
		room.HostID = ""
		for _, client := range room.Clients {
			room.HostID = client.ID
			client.IsHost = true
			log.Printf("Host left room %s. New host: %s (%s)", roomID, client.Username, client.ID)

			// Notify the new host
			hostChangedMsg := models.Message{
				Action: "HOST_CHANGED",
				Payload: mustMarshalRaw(map[string]interface{}{
					"roomId":    roomID,
					"newHostId": client.ID,
					"username":  client.Username,
				}),
			}
			SendToUser(client, mustMarshal(hostChangedMsg))
			break
		}
	}

	clientCount := len(room.Clients)
	room.Mutex.Unlock()

	// Broadcast HOST_CHANGED to all remaining clients
	if room.HostID != "" {
		hostChangedMsg := models.Message{
			Action: "HOST_CHANGED",
			Payload: mustMarshalRaw(map[string]interface{}{
				"roomId":    roomID,
				"newHostId": room.HostID,
			}),
		}
		BroadcastToRoom(room, mustMarshal(hostChangedMsg), nil)
	}

	if clientCount == 0 {
		mu.Lock()
		if r, ok := rooms[roomID]; ok {
			r.Mutex.RLock()
			stillEmpty := len(r.Clients) == 0
			r.Mutex.RUnlock()
			if stillEmpty {
				delete(rooms, roomID)
			}
		}
		mu.Unlock()
	}
}
```

#### 2c. Tambah handler `HOST_CHANGED` di frontend

Ini opsional untuk backend, tapi pastikan payload `HOST_CHANGED` yang dikirim ke frontend sudah benar dan konsisten.

### Verifikasi
```bash
cd backend && go build ./... && go run main.go

# Test 1: Buka room di 2 tab
# Tab 1 (host): buat room → harus dapat host
# Tab 2 (guest): join room → harus bukan host

# Test 2: Buka room baru, guest kirim JOIN_EVENT dengan isHost: true
# → Server harus IGNORE, guest tetap bukan host

# Test 3: Host leave room
# → Guest harus otomatis jadi host (lihat log: "New host: ...")
# → Frontend harus terima HOST_CHANGED message
```

---

## Task 3 — Fix EPISODE_ENDED Abuse (BE-C07) [KRITIS]

### Masalah
`services/message_handler.go:316-351` — `handleEpisodeEnded` tidak cek `user.IsHost`. Siapa pun bisa kirim `EPISODE_ENDED` untuk skip video berikutnya. Juga tidak ada debounce — concurrent messages bisa skip beberapa item sekaligus.

### File yang Diubah
- `backend/services/message_handler.go` — `handleEpisodeEnded`

### Implementasi

Tambah host check dan debounce:

```go
// Tambah di level package (atas file, setelah import):
var (
	episodeEndedLock sync.Map // roomID → *sync.Mutex
	lastEpisodeEnded sync.Map // roomID → time.Time
)

func handleEpisodeEnded(user *models.User, room *models.WatchRoom, payloadRaw json.RawMessage) {
	// Only host can trigger episode end (auto-advance)
	if !user.IsHost {
		log.Printf("Non-host %s tried to trigger EPISODE_ENDED", user.Username)
		return
	}

	// Debounce: prevent rapid-fire episode ended events (min 3 seconds apart)
	now := time.Now()
	if last, ok := lastEpisodeEnded.Load(room.RoomID); ok {
		if now.Sub(last.(time.Time)) < 3*time.Second {
			log.Printf("EPISODE_ENDED debounced for room %s (too soon)", room.RoomID)
			return
		}
	}
	lastEpisodeEnded.Store(room.RoomID, now)

	nextItem, err := SkipToNext(room.RoomID)
	if err != nil {
		log.Printf("Auto-advance failed (queue empty): %v", err)
		return
	}

	room.Mutex.Lock()
	room.CurrentVideo = nextItem.URL
	room.CurrentMetadata = nextItem
	room.CurrentTime = 0
	room.IsPlaying = false
	room.Mutex.Unlock()

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

	log.Printf("Auto-advanced to: %s - %s (by host %s)", nextItem.Title, nextItem.Episode, user.Username)
}
```

### Verifikasi
```bash
cd backend && go build ./... && go run main.go

# Test 1: Guest kirim EPISODE_ENDED → harus di-ignore (log: "Non-host ... tried")
# Test 2: Host kirim EPISODE_ENDED → video berikutnya diputar
# Test 3: Host kirim EPISODE_ENDED 2x dalam 1 detik → hanya 1 yang diproses (debounce)
```

---

## Task 4 — Fix WebSocket Goroutine Leak (BE-C01) [KRITIS]

### Masalah
`handlers/websocket.go:66-104` — Saat `readPump` selesai (user disconnect), `user.Conn.Close()` dipanggil tapi `user.Send` channel **tidak pernah di-close**. Akibatnya `writePump` yang melakukan `for message := range user.Send` akan block forever → goroutine leak satu per disconnect.

### File yang Diubah
- `backend/handlers/websocket.go`

### Implementasi

Gunakan `sync.Once` untuk memastikan cleanup hanya sekali, dan tutup `user.Send` untuk menghentikan `writePump`:

```go
import (
	"sync"
	// ... existing imports
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/")
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	room, exists := services.GetRoom(roomID)
	if !exists {
		services.CreateRoomWithID(roomID)
		room, _ = services.GetRoom(roomID)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade failed:", err)
		return
	}

	userID := utils.GenerateShortID()

	user := &models.User{
		ID:   userID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// cleanupOnce ensures Close + channel close happens exactly once,
	// regardless of which pump (read or write) exits first.
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(user.Send)
			user.Conn.Close()
		})
	}

	go writePump(user, cleanup)
	go readPump(user, room, cleanup)
}

func readPump(user *models.User, room *models.WatchRoom, cleanup func()) {
	defer func() {
		cleanup() // close Send channel + conn
		services.RemoveUserFromRoom(room.RoomID, user.ID)

		if user.Username != "" {
			leftMsg := models.Message{
				Action: "USER_LEFT",
				Payload: mustMarshal(models.UserLeftPayload{
					UserID:   user.ID,
					Username: user.Username,
				}),
			}
			services.BroadcastToRoom(room, mustMarshal(leftMsg), nil)
		}
	}()

	for {
		_, msgBytes, err := user.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		services.HandleMessage(user, room, msgBytes)
	}
}

func writePump(user *models.User, cleanup func()) {
	defer cleanup() // close Send channel + conn

	for message := range user.Send {
		if err := user.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}
```

### Verifikasi
```bash
cd backend && go build ./...

# Test: connect dan disconnect beberapa kali
# Periksa goroutine count tidak naik terus:
# curl http://localhost:8080/debug/pprof/goroutine?debug=1 (jika pprof diaktifkan)
# Atau: go tool pprof http://localhost:8080/debug/pprof/goroutine
```

---

## Task 5 — Fix Scrape Cache Memory Leak (BE-C06) [KRITIS]

### Masalah
`services/scraper.go:62-98` — `scrapeCache` map tidak pernah menghapus entry yang expired. Setiap entry baru ditambahkan, tapi expired entries hanya di-skip saat cache hit. Map tumbuh terus selama proses berjalan.

### File yang Diubah
- `backend/services/scraper.go`

### Implementasi

Tambahkan periodic cleanup goroutine:

```go
// Tambah setelah scrapeCacheTTL declaration (line 73):

func init() {
	// Start cache cleanup goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupScrapeCache()
		}
	}()
}

func cleanupScrapeCache() {
	scrapeCacheMu.Lock()
	defer scrapeCacheMu.Unlock()

	now := time.Now()
	before := len(scrapeCache)
	for key, entry := range scrapeCache {
		if now.After(entry.expiresAt) {
			delete(scrapeCache, key)
		}
	}
	after := len(scrapeCache)
	if before != after {
		log.Printf("[Scraper] Cache cleanup: %d → %d entries", before, after)
	}
}
```

Alternatif yang lebih baik — cleanup inline di `ScrapeStreamURLCached` (lazy cleanup):

```go
func ScrapeStreamURLCached(pageURL string) (*models.VideoMetadata, error) {
	scrapeCacheMu.RLock()
	if entry, ok := scrapeCache[pageURL]; ok && time.Now().Before(entry.expiresAt) {
		scrapeCacheMu.RUnlock()
		log.Printf("Cache hit for %s", pageURL)
		return entry.metadata, nil
	}
	scrapeCacheMu.RUnlock()

	metadata, err := ScrapeStreamURL(pageURL)
	if err != nil {
		return nil, err
	}

	scrapeCacheMu.Lock()
	scrapeCache[pageURL] = scrapeCacheEntry{
		metadata:  metadata,
		expiresAt: time.Now().Add(scrapeCacheTTL),
	}
	// Lazy cleanup: remove expired entries while we have the lock
	if len(scrapeCache) > 100 { // only cleanup when cache is large
		now := time.Now()
		for key, entry := range scrapeCache {
			if now.After(entry.expiresAt) {
				delete(scrapeCache, key)
			}
		}
	}
	scrapeCacheMu.Unlock()

	return metadata, nil
}
```

### Verifikasi
```bash
cd backend && go build ./...

# Test: scrape beberapa URL berbeda, tunggu > 30 menit (atau kurangi TTL untuk test)
# Cek log: "[Scraper] Cache cleanup: X → Y entries"
# Pastikan memory usage tidak naik terus
```

---

## Task 6 — Fix Lock Order Deadlock (BE-C02) [KRITIS]

### Masalah
`services/room_manager.go` — Ada dua jalur yang mengambil lock dengan urutan berlawanan:

**Jalur Cleanup (line 87-99):**
```go
mu.Lock()           // global lock
room.Mutex.RLock()  // room lock
```

**Jalur RemoveUserFromRoom (line 140-153):**
```go
room.Mutex.Lock()   // room lock (di line 135)
mu.Lock()           // global lock (di line 141)
```

Jika cleanup dan remove terjadi bersamaan → deadlock.

### File yang Diubah
- `backend/services/room_manager.go`

### Implementasi

Konsisten: selalu ambil `mu` (global) dulu, baru `room.Mutex`. Modifikasi `RemoveUserFromRoom`:

```go
func RemoveUserFromRoom(roomID, userID string) {
	// Ambil room reference di bawah global lock
	mu.RLock()
	room, exists := rooms[roomID]
	mu.RUnlock()
	if !exists {
		return
	}

	// Hapus user dari room
	room.Mutex.Lock()
	delete(room.Clients, userID)

	// Host reassignment (dari Task 2)
	if room.HostID == userID {
		room.HostID = ""
		for _, client := range room.Clients {
			room.HostID = client.ID
			client.IsHost = true
			log.Printf("Host left room %s. New host: %s (%s)", roomID, client.Username, client.ID)

			hostChangedMsg := models.Message{
				Action: "HOST_CHANGED",
				Payload: mustMarshalRaw(map[string]interface{}{
					"roomId":    roomID,
					"newHostId": client.ID,
					"username":  client.Username,
				}),
			}
			SendToUser(client, mustMarshal(hostChangedMsg))
			break
		}
	}

	clientCount := len(room.Clients)
	room.Mutex.Unlock()

	// Broadcast HOST_CHANGED
	if room.HostID != "" {
		hostChangedMsg := models.Message{
			Action: "HOST_CHANGED",
			Payload: mustMarshalRaw(map[string]interface{}{
				"roomId":    roomID,
				"newHostId": room.HostID,
			}),
		}
		BroadcastToRoom(room, mustMarshal(hostChangedMsg), nil)
	}

	if clientCount == 0 {
		// Konsisten: ambil global lock dulu
		mu.Lock()
		if r, ok := rooms[roomID]; ok {
			r.Mutex.RLock()
			stillEmpty := len(r.Clients) == 0
			r.Mutex.RUnlock()
			if stillEmpty {
				delete(rooms, roomID)
			}
		}
		mu.Unlock()
	}
}
```

Modifikasi `StartRoomCleanup` — konsisten urutan lock:

```go
func StartRoomCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			// Kumpulkan room IDs yang perlu di-cleanup di bawah global lock
			mu.RLock()
			var staleIDs []string
			now := time.Now()
			for id, room := range rooms {
				room.Mutex.RLock()
				isEmpty := len(room.Clients) == 0
				age := now.Sub(room.CreatedAt)
				room.Mutex.RUnlock()

				if isEmpty && age > 30*time.Minute {
					staleIDs = append(staleIDs, id)
				}
			}
			mu.RUnlock()

			// Hapus room yang stale di bawah global write lock
			if len(staleIDs) > 0 {
				mu.Lock()
				for _, id := range staleIDs {
					if r, ok := rooms[id]; ok {
						r.Mutex.RLock()
						stillEmpty := len(r.Clients) == 0
						r.Mutex.RUnlock()
						if stillEmpty {
							delete(rooms, id)
							log.Printf("Cleaned up stale room: %s", id)
						}
					}
				}
				mu.Unlock()
			}
		}
	}()
}
```

### Verifikasi
```bash
cd backend && go build ./...

# Test: jalankan Go race detector
go build -race ./...
go run -race main.go

# Buka beberapa room, join/leave secara concurrent
# Pastikan tidak ada deadlock (server tetap responsif) dan tidak ada race condition
```

---

## Task 7 — Fix Data Race di GetRoomState (BE-C08) [KRITIS]

### Masalah
`services/room_manager.go:188-206` — `GetRoomState` mengembalikan `room.Queue` (slice) dan `room.CurrentMetadata` (pointer) secara langsung. Setelah `room.Mutex.RUnlock()`, goroutine lain bisa mutate queue/metadata sementara caller masih menggunakan return value → data race.

### File yang Diubah
- `backend/services/room_manager.go`

### Implementasi

Copy data di bawah lock sebelum return:

```go
func GetRoomState(roomID string) (string, float64, bool, []models.Participant, *models.QueueItem, []*models.QueueItem) {
	room, exists := GetRoom(roomID)
	if !exists {
		return "", 0, false, nil, nil, nil
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	// Copy participants
	participants := make([]models.Participant, 0, len(room.Clients))
	for _, client := range room.Clients {
		participants = append(participants, models.Participant{
			UserID:   client.ID,
			Username: client.Username,
		})
	}

	// Copy metadata (deep copy to avoid pointer aliasing)
	var metadata *models.QueueItem
	if room.CurrentMetadata != nil {
		m := *room.CurrentMetadata
		metadata = &m
	}

	// Copy queue slice
	queueCopy := make([]*models.QueueItem, len(room.Queue))
	for i, item := range room.Queue {
		if item != nil {
			copied := *item
			queueCopy[i] = &copied
		}
	}

	return room.CurrentVideo, room.CurrentTime, room.IsPlaying, participants, metadata, queueCopy
}
```

### Verifikasi
```bash
cd backend && go build -race ./... && go run -race main.go

# Test: buka room, join, set video, add to queue
# Race detector harus clean (tidak ada WARNING: DATA RACE)
```

---

## Ringkasan Prioritas

| # | Task | Bug ID | Prioritas | Effort | File |
|---|------|--------|:---------:|:------:|------|
| 1 | SSRF Proxy Protection | BE-C04 | Kritis | Sedang | `handlers/proxy.go` |
| 2 | Host Privilege Escalation | BE-C03 | Kritis | Sedang | `message_handler.go`, `room_manager.go` |
| 3 | EPISODE_ENDED Abuse | BE-C07 | Kritis | Kecil | `message_handler.go` |
| 4 | Goroutine Leak | BE-C01 | Kritis | Kecil | `handlers/websocket.go` |
| 5 | Cache Memory Leak | BE-C06 | Kritis | Kecil | `services/scraper.go` |
| 6 | Lock Order Deadlock | BE-C02 | Kritis | Sedang | `room_manager.go` |
| 7 | Data Race GetRoomState | BE-C08 | Kritis | Kecil | `room_manager.go` |

**Catatan:** Task 2 dan 6 saling berkaitan (keduanya ubah `room_manager.go`). Disarankan kerjakan Task 6 dulu (lock order), baru Task 2 (host reassignment) karena Task 2 menambah logic di `RemoveUserFromRoom` yang harus konsisten dengan lock order yang benar.

---

*Dokumentasi ini dibuat berdasarkan analisis kode per 17 Juli 2026.*
