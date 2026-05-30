package pagination

// CursorPage is the response envelope for cursor-based pagination. It mirrors
// conventions used by Stripe, Slack, and Google AIP-158: an `items` list, a
// `meta` block exposing the next cursor + a boolean, and optional `links` for
// HATEOAS navigation.
//
// Cursor APIs are intentionally next-only here. Bidirectional cursors require
// careful repo-side support and most public APIs (Stripe, GitHub, Slack) ship
// next-only — keeping the contract narrow avoids mis-implementation.
type CursorPage[T any] struct {
	Items []T          `json:"items"`
	Meta  CursorMeta   `json:"meta"`
	Links *CursorLinks `json:"links,omitempty"`
}

// CursorMeta holds cursor-pagination metadata.
//
//   - Size         requested page size (limit echo)
//   - Count        actual number of items in this page
//   - NextCursor   opaque token for the next page; empty when none
//   - HasMore      true when NextCursor is set (matches Stripe's contract)
type CursorMeta struct {
	Size       int    `json:"size"`
	Count      int    `json:"count"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// NewCursorPage builds a CursorPage. Pass next == "" when no further page
// exists. The encoded cursor should be produced via CursorCodec.Encode.
func NewCursorPage[T any](items []T, req CursorRequest, next string) CursorPage[T] {
	if items == nil {
		items = []T{} // ensure JSON renders "items": [] not "items": null
	}
	if req.Limit < 1 {
		req.Limit = 1
	}
	return CursorPage[T]{
		Items: items,
		Meta: CursorMeta{
			Size:       req.Limit,
			Count:      len(items),
			NextCursor: next,
			HasMore:    next != "",
		},
	}
}

// WithLinks attaches navigation links and returns the page (chainable).
func (p CursorPage[T]) WithLinks(l CursorLinks) CursorPage[T] {
	p.Links = &l
	return p
}
