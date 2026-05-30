First tagged release of **go-bricks** — a collection of modular building blocks for Go. This release ships the `pagination` package.

## ✨ pagination

A production-ready, standards-aligned pagination toolkit (JSON:API · RFC 8288 · AIP-158 · Stripe-style), with **zero third-party dependencies**.

### Highlights
- **Two strategies, distinct shapes** — `OffsetPage[T]` and `CursorPage[T]`, each with its own `meta` and optional HATEOAS `links`.
- **Safe by default** — configurable `MinLimit`/`MaxLimit`/`MaxOffset`; out-of-range limits and deep offsets are rejected with typed errors; overflow-safe arithmetic; no divide-by-zero on pathological input.
- **Opaque cursors** — HMAC-SHA256 signing, TTL expiry, int64-safe decoding (`json.UseNumber`), **zero-downtime key rotation** via `RetiredSecrets`, and a decode length guard (`DefaultMaxCursorBytes`).
- **RFC 8288 Link headers** — `Links.Header()` / `CursorLinks.Header()` plus a `LinkBuilder` that preserves filters/sort while rewriting page/limit/cursor.
- **Typed sentinel errors** — `errors.Is`-friendly, with a `MapErrorToHTTPStatus` helper.
- **Well tested** — race-clean, ~94% coverage, benchmarks and runnable examples included.

### Install
```bash
go get github.com/gaguilarch93/go-bricks/pagination@v0.1.0
```

### Notes
- Pre-`v1`: the public API may still change in `v0.x` releases.
- Limits above `MaxLimit` now return `ErrInvalidLimit` (strict) rather than being silently clamped.

🤖 Generated with the help of GitHub Copilot.
