// Package pagination provides production-ready, framework-agnostic pagination
// primitives for Go services. It follows widely accepted standards:
//
//   - JSON:API / Spring HATEOAS shape for responses (items + meta + links)
//   - RFC 8288 `Link` header support for offset and cursor APIs
//   - Google AIP-158 cursor semantics (opaque, next-only by default)
//   - Stripe-style `has_more` ergonomics
//
// Two strategies are supported, each with its own response type so the API
// contract is unambiguous and OpenAPI-friendly:
//
//   - Offset pagination (page + limit): OffsetPage[T], OffsetMeta, Links
//   - Cursor pagination (opaque, optionally HMAC-signed): CursorPage[T],
//     CursorMeta, CursorLinks
//
// The package is transport-agnostic — only the standard library is imported,
// and parsing helpers operate on url.Values so it plugs into net/http, chi,
// gin, echo, fiber, gRPC-gateway, etc.
//
// Typical offset usage:
//
//	cfg := pagination.DefaultConfig()
//
//	req, err := pagination.ParseOffset(r.URL.Query(), cfg)
//	if err != nil { /* 400 */ }
//
//	items, total, err := repo.ListUsers(ctx, req.Limit, req.Offset())
//	if err != nil { /* 500 */ }
//
//	page := pagination.NewOffsetPage(items, req, total)
//	links := pagination.NewLinkBuilder(r, cfg).Offset(req, page.Meta)
//	page = page.WithLinks(links)
//
//	w.Header().Set("Link", links.Header()) // RFC 8288
//	json.NewEncoder(w).Encode(page)
package pagination
