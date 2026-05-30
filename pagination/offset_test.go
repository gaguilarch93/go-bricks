package pagination_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/gaguilarch93/go-bricks/pagination"
)

func TestDefaultConfig_Valid(t *testing.T) {
	if err := pagination.DefaultConfig().Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestConfig_Validate(t *testing.T) {
	cases := map[string]pagination.Config{
		"max<min":     {MinLimit: 5, MaxLimit: 1, DefaultLimit: 5},
		"default OOB": {MinLimit: 1, MaxLimit: 10, DefaultLimit: 20},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if err := c.Validate(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestConfig_NegativeMaxOffsetDisablesCap(t *testing.T) {
	cfg := pagination.DefaultConfig()
	cfg.MaxOffset = -1
	if err := cfg.Validate(); err != nil {
		t.Fatalf("negative MaxOffset should be valid (means no cap): %v", err)
	}
	// A huge offset that would normally fail must now succeed.
	if _, err := pagination.ParseOffset(url.Values{"offset": {"100000000"}, "limit": {"10"}}, cfg); err != nil {
		t.Fatalf("expected no cap, got %v", err)
	}
}

func TestParseOffset_Defaults(t *testing.T) {
	cfg := pagination.DefaultConfig()
	req, err := pagination.ParseOffset(url.Values{}, cfg)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if req.Page != 1 || req.Limit != cfg.DefaultLimit || req.Offset() != 0 {
		t.Fatalf("got %+v", req)
	}
}

func TestParseOffset_PageAndLimit(t *testing.T) {
	cfg := pagination.DefaultConfig()
	v := url.Values{"page": {"3"}, "limit": {"25"}}
	req, err := pagination.ParseOffset(v, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if req.Page != 3 || req.Limit != 25 || req.Offset() != 50 {
		t.Fatalf("got %+v", req)
	}
}

func TestParseOffset_OffsetConverts(t *testing.T) {
	cfg := pagination.DefaultConfig()
	v := url.Values{"offset": {"40"}, "limit": {"10"}}
	req, err := pagination.ParseOffset(v, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if req.Page != 5 || req.Limit != 10 {
		t.Fatalf("got %+v", req)
	}
}

func TestParseOffset_LimitAboveMaxRejected(t *testing.T) {
	cfg := pagination.DefaultConfig()
	v := url.Values{"limit": {"9999"}}
	_, err := pagination.ParseOffset(v, cfg)
	if !errors.Is(err, pagination.ErrInvalidLimit) {
		t.Fatalf("expected ErrInvalidLimit for limit above MaxLimit, got %v", err)
	}
}

func TestParseOffset_InvalidInputs(t *testing.T) {
	cfg := pagination.DefaultConfig()
	cases := map[string]struct {
		v       url.Values
		errKind error
	}{
		"non-int limit": {url.Values{"limit": {"abc"}}, pagination.ErrInvalidLimit},
		"zero limit":    {url.Values{"limit": {"0"}}, pagination.ErrInvalidLimit},
		"neg limit":     {url.Values{"limit": {"-3"}}, pagination.ErrInvalidLimit},
		"zero page":     {url.Values{"page": {"0"}}, pagination.ErrInvalidPage},
		"neg page":      {url.Values{"page": {"-1"}}, pagination.ErrInvalidPage},
		"both":          {url.Values{"page": {"1"}, "offset": {"0"}}, pagination.ErrInvalidPage},
		"neg offset":    {url.Values{"offset": {"-1"}}, pagination.ErrInvalidOffset},
		"too deep":      {url.Values{"offset": {"99999999"}}, pagination.ErrInvalidOffset},
		"misaligned":    {url.Values{"offset": {"15"}, "limit": {"10"}}, pagination.ErrInvalidOffset},
		"overflow page": {url.Values{"page": {"99999999999999"}, "limit": {"100"}}, pagination.ErrInvalidOffset},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := pagination.ParseOffset(tc.v, cfg)
			if !errors.Is(err, tc.errKind) {
				t.Fatalf("expected %v, got %v", tc.errKind, err)
			}
		})
	}
}

func TestParseOffset_CustomParams(t *testing.T) {
	cfg := pagination.DefaultConfig()
	cfg.PageParam = "p"
	cfg.LimitParam = "n"
	req, err := pagination.ParseOffset(url.Values{"p": {"2"}, "n": {"5"}}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if req.Page != 2 || req.Limit != 5 {
		t.Fatalf("got %+v", req)
	}
}

func TestNewOffsetPage_WithTotal(t *testing.T) {
	req := pagination.OffsetRequest{Page: 2, Limit: 10}
	items := []int{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	p := pagination.NewOffsetPage(items, req, 35)
	if p.Meta.Total == nil || *p.Meta.Total != 35 {
		t.Fatal("total")
	}
	if p.Meta.TotalPages == nil || *p.Meta.TotalPages != 4 {
		t.Fatalf("total_pages got %v", p.Meta.TotalPages)
	}
	if !p.Meta.HasMore {
		t.Fatal("expected has_more")
	}
	if p.Meta.Page != 2 || p.Meta.Size != 10 || p.Meta.Count != 10 {
		t.Fatalf("meta=%+v", p.Meta)
	}
	if p.Links != nil {
		t.Fatal("links should be nil until WithLinks is called")
	}
}

func TestNewOffsetPage_UnknownTotal(t *testing.T) {
	req := pagination.OffsetRequest{Page: 1, Limit: 5}
	p := pagination.NewOffsetPage([]int{1, 2, 3, 4, 5}, req, -1)
	if p.Meta.Total != nil {
		t.Fatal("total should be nil")
	}
	if !p.Meta.HasMore {
		t.Fatal("expected has_more (full page)")
	}
}

func TestNewOffsetPage_LastPage(t *testing.T) {
	req := pagination.OffsetRequest{Page: 4, Limit: 10}
	p := pagination.NewOffsetPage([]int{31, 32, 33, 34, 35}, req, 35)
	if p.Meta.HasMore {
		t.Fatal("expected no more")
	}
	if p.Meta.Count != 5 || p.Meta.Size != 10 {
		t.Fatalf("count/size mismatch: %+v", p.Meta)
	}
}

func TestNewOffsetPage_DefensiveClamps(t *testing.T) {
	// Pathological request must not panic and must produce sane meta.
	bad := pagination.OffsetRequest{Page: 0, Limit: 0}
	p := pagination.NewOffsetPage([]int{}, bad, -1)
	if p.Meta.Page != 1 || p.Meta.Size != 1 {
		t.Fatalf("expected clamp to (1,1), got %+v", p.Meta)
	}
}

func TestOffsetRequest_OffsetGuards(t *testing.T) {
	if (pagination.OffsetRequest{Page: 0, Limit: 10}).Offset() != 0 {
		t.Fatal("expected 0 for Page<1")
	}
	if (pagination.OffsetRequest{Page: 5, Limit: 0}).Offset() != 0 {
		t.Fatal("expected 0 for Limit<1")
	}
}

func TestZeroValueConfig_Works(t *testing.T) {
	// A zero Config should behave like DefaultConfig via withDefaults().
	req, err := pagination.ParseOffset(url.Values{}, pagination.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if req.Limit != 20 || req.Page != 1 {
		t.Fatalf("got %+v", req)
	}
}

func TestNewOffsetPage_NilItemsBecomeEmpty(t *testing.T) {
	req := pagination.OffsetRequest{Page: 1, Limit: 10}
	p := pagination.NewOffsetPage[int](nil, req, 0)
	if p.Items == nil {
		t.Fatal("items should be non-nil empty slice, not nil")
	}
	if len(p.Items) != 0 {
		t.Fatalf("expected empty slice, got %v", p.Items)
	}
	// JSON encoding sanity.
	b, _ := json.Marshal(p)
	if !strings.Contains(string(b), `"items":[]`) {
		t.Fatalf("expected items:[] in JSON, got %s", b)
	}
}

func TestZeroValueConfig_DefaultLimitClampedToBounds(t *testing.T) {
	// User raised MinLimit but left DefaultLimit unset → DefaultLimit must
	// be clamped up to MinLimit, not left at the library default of 20.
	cfg := pagination.Config{MinLimit: 50, MaxLimit: 100}
	req, err := pagination.ParseOffset(url.Values{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if req.Limit < 50 {
		t.Fatalf("expected limit >= MinLimit (50), got %d", req.Limit)
	}
}

func TestMapErrorToHTTPStatus(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 200},
		{pagination.ErrInvalidLimit, 400},
		{pagination.ErrInvalidPage, 400},
		{pagination.ErrInvalidCursor, 400},
		{pagination.ErrCursorTamper, 400},
		{pagination.ErrCursorExpired, 400},
		{errors.New("boom"), 500},
	}
	for _, tc := range cases {
		if got := pagination.MapErrorToHTTPStatus(tc.err); got != tc.want {
			t.Fatalf("err=%v want=%d got=%d", tc.err, tc.want, got)
		}
	}
}
