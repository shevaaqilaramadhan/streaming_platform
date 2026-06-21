# Backend Fix Required — WebSocket EPIPE / ECONNRESET

## Problem

The Vite dev server (port `5174`) proxies WebSocket connections to the Go backend (port `8080`).
When the proxy forwards a connection, the `Origin` header reads `http://localhost:5174`.

Gorilla WebSocket's **default `CheckOrigin`** rejects any request where `Origin` doesn't match the `Host` header.
Since `Host` is `localhost:8080` but `Origin` is `localhost:5174` → the Go server **immediately closes the connection** → Vite logs `EPIPE` / `ECONNRESET`.

```
Browser ──► ws://localhost:5174/ws/roomId
                │
         Vite proxy forwards
                │
Go backend ◄── ws://localhost:8080/ws/roomId
                   Origin: localhost:5174  ← mismatch with Host: localhost:8080
                   ✗  CheckOrigin fails → connection closed → EPIPE
```

---

## Fix — `backend/handlers/websocket.go`

Find your `websocket.Upgrader` declaration and add `CheckOrigin`:

```go
// ❌ BEFORE — default CheckOrigin rejects the proxied origin
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}
```

```go
// ✅ AFTER — allow all origins (fine for dev; restrict in production)
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}
```

> **Note for production:** Replace `return true` with an allowlist check, e.g.:
> ```go
> CheckOrigin: func(r *http.Request) bool {
>     origin := r.Header.Get("Origin")
>     return origin == "https://your-production-domain.com"
> },
> ```

---

## What the frontend already fixed

`vite.config.js` was updated with `rewriteWsOrigin: true`, which rewrites the `Origin` header on proxied WS connections to match the target host (`localhost:8080`). This alone may be enough — but adding the `CheckOrigin` override on the Go side is the robust fix that also works in production and direct connections (not via Vite proxy).

---

## After applying the fix

Restart the Go server:
```bash
# in backend/
go run main.go
```

Errors in the Vite terminal should stop immediately.
