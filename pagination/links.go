package pagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Links holds HATEOAS navigation URLs for offset pagination, following the
// JSON:API / Spring HATEOAS / GitHub Link header conventions (self, first,
// prev, next, last).
type Links struct {
	Self  string `json:"self,omitempty"`
	First string `json:"first,omitempty"`
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	Last  string `json:"last,omitempty"`
}

// CursorLinks holds HATEOAS navigation URLs for cursor pagination. Only
// `self` and `next` are exposed — cursor APIs are next-only by design.
type CursorLinks struct {
	Self string `json:"self,omitempty"`
	Next string `json:"next,omitempty"`
}

// Header returns the value for an HTTP `Link` header per RFC 8288. Empty
// fields are skipped; returns "" when no links are set.
func (l Links) Header() string {
	parts := make([]string, 0, 4)
	parts = appendLink(parts, l.First, "first")
	parts = appendLink(parts, l.Prev, "prev")
	parts = appendLink(parts, l.Next, "next")
	parts = appendLink(parts, l.Last, "last")
	return strings.Join(parts, ", ")
}

// Header returns the RFC 8288 `Link` header value (next-only).
func (l CursorLinks) Header() string {
	parts := appendLink(nil, l.Next, "next")
	return strings.Join(parts, ", ")
}

// %q produces a Go-quoted string, which for the ASCII rel-types used here
// ("first"/"prev"/"next"/"last") matches the RFC 8288 quoted-string form
// exactly. Custom rels containing control chars or non-ASCII would need
// stricter handling, but those values are not produced by this package.
func appendLink(parts []string, urlStr, rel string) []string {
	if urlStr == "" {
		return parts
	}
	return append(parts, fmt.Sprintf("<%s>; rel=%q", urlStr, rel))
}

// LinkBuilder constructs navigation URLs that preserve non-pagination query
// parameters (filters, sort, etc.) while rewriting page/limit/cursor.
//
// Construct via NewLinkBuilder for the common case of building from an
// *http.Request (relative URLs based on r.URL.Path). For absolute URLs (e.g.
// to satisfy mobile clients or downstream API gateways), set BaseURL directly:
//
//	lb := pagination.NewLinkBuilder(r, cfg)
//	lb.BaseURL = "https://api.example.com" + r.URL.Path
type LinkBuilder struct {
	BaseURL   string     // path or absolute URL, e.g. "/v1/users" or "https://api.example.com/v1/users"
	BaseQuery url.Values // existing query without pagination params
	Config    Config
}

// NewLinkBuilder derives a LinkBuilder from an HTTP request, stripping the
// pagination parameters from the base query so they can be re-applied per
// link without duplication.
func NewLinkBuilder(r *http.Request, cfg Config) LinkBuilder {
	c := cfg.withDefaults()
	q := cloneValues(r.URL.Query())
	q.Del(c.LimitParam)
	q.Del(c.PageParam)
	q.Del(c.OffsetParam)
	q.Del(c.CursorParam)
	return LinkBuilder{BaseURL: r.URL.Path, BaseQuery: q, Config: c}
}

// Offset builds the navigation set for an offset-paginated response.
// Pass the meta returned by NewOffsetPage so first/last/next are accurate.
func (lb LinkBuilder) Offset(req OffsetRequest, meta OffsetMeta) Links {
	c := lb.Config.withDefaults()
	build := func(page int) string {
		q := cloneValues(lb.BaseQuery)
		q.Set(c.PageParam, strconv.Itoa(page))
		q.Set(c.LimitParam, strconv.Itoa(req.Limit))
		return joinURL(lb.BaseURL, q)
	}
	links := Links{
		Self:  build(req.Page),
		First: build(1),
	}
	if req.Page > 1 {
		links.Prev = build(req.Page - 1)
	}
	if meta.HasMore {
		links.Next = build(req.Page + 1)
	}
	if meta.TotalPages != nil {
		links.Last = build(*meta.TotalPages)
	}
	return links
}

// Cursor builds the navigation set for a cursor-paginated response.
// `nextCursor` should be the same opaque token written into CursorMeta.
func (lb LinkBuilder) Cursor(req CursorRequest, nextCursor string) CursorLinks {
	c := lb.Config.withDefaults()
	build := func(cursor string) string {
		q := cloneValues(lb.BaseQuery)
		q.Set(c.LimitParam, strconv.Itoa(req.Limit))
		if cursor != "" {
			q.Set(c.CursorParam, cursor)
		}
		return joinURL(lb.BaseURL, q)
	}
	links := CursorLinks{Self: build(req.Cursor)}
	if nextCursor != "" {
		links.Next = build(nextCursor)
	}
	return links
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vs := range v {
		cp := make([]string, len(vs))
		copy(cp, vs)
		out[k] = cp
	}
	return out
}

func joinURL(base string, q url.Values) string {
	if len(q) == 0 {
		return base
	}
	// Respect a BaseURL that already carries query parameters (e.g.
	// "/v1/users?env=prod") or a fragment.
	sep := "?"
	if i := strings.IndexByte(base, '?'); i >= 0 {
		// If the existing query is empty ("/v1/users?"), keep "?"; otherwise
		// append with "&".
		if i < len(base)-1 {
			sep = "&"
		} else {
			sep = ""
		}
	}
	return base + sep + q.Encode()
}
