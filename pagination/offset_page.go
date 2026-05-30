package pagination

// OffsetPage is the response envelope for offset-based pagination. It is
// shaped after JSON:API and Spring HATEOAS conventions: a list of items, a
// `meta` object with page info, and an optional `links` object for HATEOAS
// navigation. Use a *LinkBuilder* to populate Links.
type OffsetPage[T any] struct {
	Items []T        `json:"items"`
	Meta  OffsetMeta `json:"meta"`
	Links *Links     `json:"links,omitempty"`
}

// OffsetMeta holds offset-pagination metadata.
//
//   - Page         current 1-based page number
//   - Size         requested page size (limit echo)
//   - Count        actual number of items in this page (<= Size on the last page)
//   - Total        total matching records; omitted when unknown
//   - TotalPages   ceil(Total / Size); omitted when Total is unknown
//   - HasMore      true when at least one more page exists
type OffsetMeta struct {
	Page       int    `json:"page"`
	Size       int    `json:"size"`
	Count      int    `json:"count"`
	Total      *int64 `json:"total,omitempty"`
	TotalPages *int   `json:"total_pages,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// NewOffsetPage builds an OffsetPage. Pass `total = -1` when the total is
// unknown or too expensive to compute; Total/TotalPages will be omitted and
// HasMore will fall back to a "full page" heuristic.
//
// The function defends against pathological requests (Page < 1 or Limit < 1)
// by clamping locally so a misuse never panics with divide-by-zero.
func NewOffsetPage[T any](items []T, req OffsetRequest, total int64) OffsetPage[T] {
	if items == nil {
		items = []T{} // ensure JSON renders "items": [] not "items": null
	}
	if req.Limit < 1 {
		req.Limit = 1
	}
	if req.Page < 1 {
		req.Page = 1
	}
	meta := OffsetMeta{
		Page:  req.Page,
		Size:  req.Limit,
		Count: len(items),
	}
	if total >= 0 {
		t := total
		meta.Total = &t
		tp := int((total + int64(req.Limit) - 1) / int64(req.Limit))
		if tp < 1 {
			tp = 1
		}
		meta.TotalPages = &tp
		meta.HasMore = int64(req.Offset()+len(items)) < total
	} else {
		meta.HasMore = len(items) >= req.Limit
	}
	return OffsetPage[T]{Items: items, Meta: meta}
}

// WithLinks attaches navigation links and returns the page (chainable).
func (p OffsetPage[T]) WithLinks(l Links) OffsetPage[T] {
	p.Links = &l
	return p
}
