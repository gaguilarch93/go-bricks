package pagination_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaguilarch93/go-bricks/pagination"
)

func TestLinkBuilder_Offset_PreservesFilters(t *testing.T) {
	cfg := pagination.DefaultConfig()
	r := httptest.NewRequest("GET", "/v1/users?status=active&sort=name&page=2&limit=10", nil)
	lb := pagination.NewLinkBuilder(r, cfg)

	req := pagination.OffsetRequest{Page: 2, Limit: 10}
	page := pagination.NewOffsetPage([]int{}, req, 45) // 5 pages total
	links := lb.Offset(req, page.Meta)

	mustContain := func(s, sub string) {
		t.Helper()
		if !strings.Contains(s, sub) {
			t.Fatalf("expected %q in %q", sub, s)
		}
	}
	mustContain(links.Self, "/v1/users?")
	mustContain(links.Self, "status=active")
	mustContain(links.Self, "sort=name")
	mustContain(links.Self, "page=2")
	mustContain(links.Self, "limit=10")
	mustContain(links.First, "page=1")
	mustContain(links.Prev, "page=1")
	mustContain(links.Next, "page=3")
	mustContain(links.Last, "page=5")
}

func TestLinkBuilder_Offset_FirstPageHasNoPrev(t *testing.T) {
	cfg := pagination.DefaultConfig()
	r := httptest.NewRequest("GET", "/x", nil)
	lb := pagination.NewLinkBuilder(r, cfg)
	req := pagination.OffsetRequest{Page: 1, Limit: 10}
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	p := pagination.NewOffsetPage(items, req, 10)
	links := lb.Offset(req, p.Meta)
	if links.Prev != "" {
		t.Fatalf("prev should be empty on first page: %q", links.Prev)
	}
	if links.Next != "" {
		t.Fatalf("next should be empty on last page: %q", links.Next)
	}
}

func TestLinkBuilder_Offset_UnknownTotal_NoLast(t *testing.T) {
	cfg := pagination.DefaultConfig()
	r := httptest.NewRequest("GET", "/x", nil)
	lb := pagination.NewLinkBuilder(r, cfg)
	req := pagination.OffsetRequest{Page: 1, Limit: 5}
	p := pagination.NewOffsetPage([]int{1, 2, 3, 4, 5}, req, -1) // unknown
	links := lb.Offset(req, p.Meta)
	if links.Last != "" {
		t.Fatalf("last should be empty when total unknown: %q", links.Last)
	}
	if links.Next == "" {
		t.Fatal("next should be set (full page heuristic)")
	}
}

func TestLinkBuilder_Cursor(t *testing.T) {
	cfg := pagination.DefaultConfig()
	r := httptest.NewRequest("GET", "/v1/events?type=signup&cursor=cur1&limit=5", nil)
	lb := pagination.NewLinkBuilder(r, cfg)
	req := pagination.CursorRequest{Cursor: "cur1", Limit: 5}
	links := lb.Cursor(req, "cur2")
	if !strings.Contains(links.Self, "type=signup") {
		t.Fatalf("filter dropped: %q", links.Self)
	}
	if !strings.Contains(links.Self, "cursor=cur1") {
		t.Fatalf("self cursor: %q", links.Self)
	}
	if !strings.Contains(links.Next, "cursor=cur2") {
		t.Fatalf("next cursor: %q", links.Next)
	}

	// No next when next cursor is empty.
	links = lb.Cursor(req, "")
	if links.Next != "" {
		t.Fatalf("next should be empty: %q", links.Next)
	}
}

func TestLinks_Header_RFC8288(t *testing.T) {
	l := pagination.Links{
		First: "/x?page=1",
		Prev:  "/x?page=2",
		Next:  "/x?page=4",
		Last:  "/x?page=10",
	}
	h := l.Header()
	for _, sub := range []string{
		`</x?page=1>; rel="first"`,
		`</x?page=2>; rel="prev"`,
		`</x?page=4>; rel="next"`,
		`</x?page=10>; rel="last"`,
	} {
		if !strings.Contains(h, sub) {
			t.Fatalf("missing %q in %q", sub, h)
		}
	}

	if (pagination.Links{}).Header() != "" {
		t.Fatal("empty links should produce empty header")
	}
}

func TestLinkBuilder_BaseURLWithExistingQuery(t *testing.T) {
	// Manually set BaseURL with a pre-existing query (use case: API gateway
	// stripped some params or caller composes the absolute URL manually).
	cfg := pagination.DefaultConfig()
	lb := pagination.LinkBuilder{
		BaseURL:   "/v1/users?env=prod",
		BaseQuery: nil,
		Config:    cfg,
	}
	req := pagination.OffsetRequest{Page: 2, Limit: 10}
	p := pagination.NewOffsetPage([]int{}, req, 100)
	links := lb.Offset(req, p.Meta)
	// Must use & to append, not a second ?.
	if strings.Count(links.Self, "?") != 1 {
		t.Fatalf("expected single ? in URL, got %q", links.Self)
	}
	if !strings.Contains(links.Self, "env=prod") {
		t.Fatalf("existing query dropped: %q", links.Self)
	}
	if !strings.Contains(links.Self, "page=2") {
		t.Fatalf("pagination params not appended: %q", links.Self)
	}
}

func TestLinkBuilder_BaseURLWithTrailingQuestion(t *testing.T) {
	cfg := pagination.DefaultConfig()
	lb := pagination.LinkBuilder{BaseURL: "/v1/users?", Config: cfg}
	req := pagination.OffsetRequest{Page: 1, Limit: 10}
	p := pagination.NewOffsetPage([]int{}, req, 1)
	links := lb.Offset(req, p.Meta)
	if strings.Count(links.Self, "?") != 1 {
		t.Fatalf("expected single ? in URL, got %q", links.Self)
	}
}

func TestCursorLinks_Header_RFC8288(t *testing.T) {
	l := pagination.CursorLinks{Next: "/x?cursor=abc"}
	if l.Header() != `</x?cursor=abc>; rel="next"` {
		t.Fatalf("got %q", l.Header())
	}
	if (pagination.CursorLinks{}).Header() != "" {
		t.Fatal("empty cursor links should produce empty header")
	}
}
