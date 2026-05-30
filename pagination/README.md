# pagination

Production-ready, framework-agnostic pagination for Go. Aligned with widely
accepted standards:

- **JSON:API / Spring HATEOAS** response shape — `items` + `meta` + `links`
- **RFC 8288** `Link` header support for both strategies
- **Google AIP-158** cursor semantics (opaque, next-only)
- **Stripe-style `has_more`** ergonomics

## Features

- Two distinct response types per strategy — clean OpenAPI contract:
  - `OffsetPage[T]` + `OffsetMeta` + `Links` (self / first / prev / next / last)
  - `CursorPage[T]` + `CursorMeta` + `CursorLinks` (self / next)
- **Safe by default**: out-of-range limits rejected against `MinLimit`/`MaxLimit`, deep offsets rejected via `MaxOffset`, typed sentinel errors (use `errors.Is`).
- **Opaque cursors** with optional HMAC-SHA256 signing and TTL — prevents tampering and forgery.
- **HATEOAS link builder** that preserves filters/sort while rewriting pagination params.
- **Zero third-party dependencies** — standard library only.

## Install

```bash
go get github.com/gaguilarch93/go-bricks/pagination
```

## Offset pagination

```go
cfg := pagination.DefaultConfig() // limit=20, max=100, max_offset=10_000

func listUsers(w http.ResponseWriter, r *http.Request) {
    req, err := pagination.ParseOffset(r.URL.Query(), cfg)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    users, total, err := repo.ListUsers(r.Context(), req.Limit, req.Offset())
    if err != nil { /* 500 */ }

    page  := pagination.NewOffsetPage(users, req, total)
    links := pagination.NewLinkBuilder(r, cfg).Offset(req, page.Meta)
    page   = page.WithLinks(links)

    w.Header().Set("Link", links.Header()) // RFC 8288
    json.NewEncoder(w).Encode(page)
}
```

Response body:

```json
{
  "items": [...],
  "meta": {
    "page": 2,
    "size": 20,
    "count": 20,
    "total": 137,
    "total_pages": 7,
    "has_more": true
  },
  "links": {
    "self":  "/v1/users?limit=20&page=2&status=active",
    "first": "/v1/users?limit=20&page=1&status=active",
    "prev":  "/v1/users?limit=20&page=1&status=active",
    "next":  "/v1/users?limit=20&page=3&status=active",
    "last":  "/v1/users?limit=20&page=7&status=active"
  }
}
```

Pass `total = -1` to `NewOffsetPage` when you can't (or won't) compute a count.
`total` / `total_pages` / `last` are then omitted and `has_more` falls back to
the "full page" heuristic.

## Cursor pagination

```go
codec := pagination.CursorCodec{
    Secret: []byte(os.Getenv("CURSOR_SECRET")), // HMAC-SHA256 signing
    TTL:    24 * time.Hour,                     // optional expiry
}

func listEvents(w http.ResponseWriter, r *http.Request) {
    req, err := pagination.ParseCursor(r.URL.Query(), cfg)
    if err != nil { /* 400 */ }

    var after pagination.CursorPayload
    if req.Cursor != "" {
        after, err = codec.Decode(req.Cursor)
        if err != nil { /* 400 — invalid / tampered / expired */ }
    }

    // Fetch limit+1 to detect "more"; truncate before responding.
    events, hasMore, err := repo.ListEvents(r.Context(), after, req.Limit)
    if err != nil { /* 500 */ }

    var next string
    if hasMore {
        last := events[len(events)-1]
        next, _ = codec.Encode(pagination.CursorPayload{
            "created_at": last.CreatedAt,
            "id":         last.ID,
        })
    }

    page  := pagination.NewCursorPage(events, req, next)
    links := pagination.NewLinkBuilder(r, cfg).Cursor(req, next)
    page   = page.WithLinks(links)

    w.Header().Set("Link", links.Header())
    json.NewEncoder(w).Encode(page)
}
```

**Key rotation (zero downtime):** to change `Secret` without invalidating
in-flight cursors, move the old key into `RetiredSecrets` (verify-only) and set
a fresh `Secret`:

```go
codec := pagination.CursorCodec{
    Secret:         []byte(os.Getenv("CURSOR_SECRET_V2")),      // signs new cursors
    RetiredSecrets: [][]byte{[]byte(os.Getenv("CURSOR_SECRET_V1"))}, // still verify old ones
    TTL:            24 * time.Hour,
}
```

Cursors signed with a retired key keep decoding until they expire; drop the
retired key once the TTL window has passed.

**Length guard:** `Decode` rejects tokens longer than `MaxCursorBytes`
(default `DefaultMaxCursorBytes` = 8 KiB) to bound decode allocations.

Response body:

```json
{
  "items": [...],
  "meta": {
    "size": 20,
    "count": 20,
    "next_cursor": "eyJ2IjoxLCJ0Ijox...",
    "has_more": true
  },
  "links": {
    "self": "/v1/events?cursor=eyJ...prev&limit=20&type=signup",
    "next": "/v1/events?cursor=eyJ...next&limit=20&type=signup"
  }
}
```

## Configuration

| Field          | Default  | Purpose                                                                 |
|----------------|----------|-------------------------------------------------------------------------|
| `DefaultLimit` | `20`     | Applied when `limit` is omitted. (`0` → use default.)                   |
| `MinLimit`     | `1`      | Reject limits below this. (`0` → use default.)                          |
| `MaxLimit`     | `100`    | Reject limits above this. (`0` → use default.)                          |
| `MaxOffset`    | `10000`  | Reject deep offsets. `0` → use default; **negative → cap disabled**.    |
| `LimitParam`   | `limit`  | Query key for limit.                                                    |
| `PageParam`    | `page`   | Query key for page.                                                     |
| `OffsetParam`  | `offset` | Query key for offset (alternative to page; must be a multiple of limit).|
| `CursorParam`  | `cursor` | Query key for cursor.                                                   |

A zero-value `Config{}` is usable — missing fields are filled from
`DefaultConfig()` at parse time. Call `cfg.Validate()` at startup to catch
explicit misconfigurations early.

## Errors

All parsing/decoding errors wrap one of:

- `ErrInvalidLimit`, `ErrInvalidPage`, `ErrInvalidOffset`
- `ErrInvalidCursor`, `ErrCursorTamper`, `ErrCursorExpired`
- `ErrCursorConfig` (e.g. `TTL` set without `Secret`)

Use `errors.Is` to branch, or the bundled helper:

```go
if err != nil {
    http.Error(w, err.Error(), pagination.MapErrorToHTTPStatus(err))
    return
}
```

## Cursor payload number handling

JSON has no integer type, so `encoding/json` decodes numbers as `float64`,
corrupting IDs above 2^53. To preserve precision, `CursorCodec.Decode` uses a
`json.Decoder` with `UseNumber()` — numeric fields arrive as `json.Number`:

```go
payload, _ := codec.Decode(req.Cursor)
id, _ := payload["id"].(json.Number).Int64()
```

## Design notes

- **Separate page types** instead of a unified `Page[T]` keeps the API
  contract explicit and prevents impossible field combinations in
  OpenAPI / generated clients.
- **HATEOAS links are opt-in** (`*Links` pointer) so consumers that don't need
  them get a smaller payload and zero allocation overhead.
- **Cursor APIs are next-only** by default (matches Stripe / GitHub / Slack /
  AIP-158). Bidirectional cursors require careful repo support and are easy
  to mis-implement; we don't ship a footgun.
- **Cursor payloads should be small and stable** (sort key + tiebreaker).
  Never put filters or user-controlled state in them.
- **Sign your cursors in public APIs.** Unsigned cursors are fine for trusted
  internal traffic but let anyone craft pointers into arbitrary internal
  state.
