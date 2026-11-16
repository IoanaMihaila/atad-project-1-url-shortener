# Architecture Note — URL Shortener with Analytics

## The shape

The project is a single Go HTTP service using the standard `net/http` package, listening on port `8080` inside a Docker container.

Requests enter through `cmd/app/main.go` and are routed to:
- The four common contract endpoints (`/health`, `/version`, `/`, `/reset`).
- The URL shortener endpoints (`POST /links`, `GET /links`, `GET /links/{code}/stats`, `DELETE /links/{code}`, and `GET /{code}`).

On `GET /{code}`, the handler looks up the short code in memory, increments its visit counter, updates `last_visit_at`, persists the change, and returns a `302 Found` redirect with the `Location` header set to the original URL.

## The data

Each link is stored as a Go struct with the fields required by the specification:
- `id`: auto-incrementing integer starting at `1` (resets to `1` on `POST /reset`)
- `code`: 4–10 characters (`A-Z`, `a-z`, `0-9`, `_`), or `L = 7` characters when generated randomly
- `url`: validated `http://` or `https://` URL
- `created_at`: ISO 8601 timestamp
- `visits`: integer counter starting at `0`
- `last_visit_at`: ISO 8601 timestamp, `null` until the first visit

While running, links are stored in a map indexed by `code` (plus an ordered slice or ID tracking for listing newest first), guarded by a `sync.RWMutex`. To survive container restarts, the state is saved to a JSON file on disk (written atomically via temp file + rename) whenever links are created, visited, deleted, or reset, and loaded back at startup.

## Three decisions

1. **Go instead of Rust:**
   I chose Go because its standard library (`net/http`, `encoding/json`, `sync`) lets me build and test an HTTP service much faster without spending weeks fighting ownership rules when sharing state across concurrent handlers.

2. **Updating click analytics synchronously before returning `302`:**
   I update `visits` and `last_visit_at` inside the `GET /{code}` request path rather than offloading to a background goroutine/channel. A background worker would reduce redirect latency, but a crash or container stop could lose recent clicks. Updating synchronously is simpler and guarantees every served redirect is recorded.

3. **Checking and inserting codes under a single mutex lock:**
   To handle simultaneous requests for the same custom code (`POST /links`), I check if the code exists and insert it inside the same `mu.Lock()` critical section (and retry random generation up to a few times if a 7-character code collides). The alternative was checking with a read lock first and upgrading to a write lock, which would leave a race window where two requests could pass the check at the same time.

## What is not built yet

Right now, I have configured `atad.json`, the lowercase GHCR image tag in `.github/workflows/ci.yml`, `README.md`, and the landing page in `cmd/app/main.go`.

What remains to be implemented:
- Storage layer in `internal/shortener` (in-memory map, mutex synchronization, and JSON file persistence across restarts).
- `POST /links` with `http`/`https` validation, custom code rules, and `L = 7` random code generation.
- `GET /{code}` redirect + visit tracking, `GET /links` (newest first), `GET /links/{code}/stats`, and `DELETE /links/{code}`.
- Connecting `POST /reset` to clear both memory and the persisted file, plus unit and concurrency tests.