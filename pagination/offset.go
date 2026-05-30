package pagination

import (
	"fmt"
	"net/url"
	"strconv"
)

// OffsetRequest is a validated offset-pagination request. Always construct via
// ParseOffset or supply Page >= 1 and Limit >= 1 manually; the Offset() helper
// guards against bad values but other consumers may not.
type OffsetRequest struct {
	Page  int
	Limit int
}

// Offset returns the SQL/DB offset implied by Page and Limit. Returns 0 for
// any Page < 1 or Limit < 1 to keep callers from issuing negative offsets.
func (r OffsetRequest) Offset() int {
	if r.Page < 1 || r.Limit < 1 {
		return 0
	}
	return (r.Page - 1) * r.Limit
}

// ParseOffset reads and validates an OffsetRequest from query parameters,
// applying the Config's defaults and bounds. It accepts either:
//
//	?page=2&limit=50
//	?offset=50&limit=50   (must be a multiple of limit)
//
// Non-aligned offsets (e.g. offset=15&limit=10) are rejected with
// ErrInvalidOffset to keep page/offset/links self-consistent — use
// ?page=N&limit=M instead when you can't align.
//
// Missing values fall back to defaults. Out-of-range limits (below MinLimit or
// above MaxLimit) return an error wrapping ErrInvalidLimit; other invalid
// (non-integer, negative) values likewise return a sentinel-wrapped error.
func ParseOffset(values url.Values, cfg Config) (OffsetRequest, error) {
	cfg = cfg.withDefaults()

	limit, err := parseLimit(values.Get(cfg.LimitParam), cfg)
	if err != nil {
		return OffsetRequest{}, err
	}

	pageStr := values.Get(cfg.PageParam)
	offsetStr := values.Get(cfg.OffsetParam)

	switch {
	case pageStr != "" && offsetStr != "":
		return OffsetRequest{}, fmt.Errorf("%w: specify either %q or %q, not both",
			ErrInvalidPage, cfg.PageParam, cfg.OffsetParam)

	case pageStr != "":
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return OffsetRequest{}, fmt.Errorf("%w: %q is not a positive integer", ErrInvalidPage, pageStr)
		}
		// Overflow-safe equivalent of (page-1)*limit > MaxOffset.
		if cfg.MaxOffset > 0 && page-1 > cfg.MaxOffset/limit {
			return OffsetRequest{}, fmt.Errorf("%w: page %d with limit %d exceeds MaxOffset %d",
				ErrInvalidOffset, page, limit, cfg.MaxOffset)
		}
		return OffsetRequest{Page: page, Limit: limit}, nil

	case offsetStr != "":
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return OffsetRequest{}, fmt.Errorf("%w: %q is not a non-negative integer", ErrInvalidOffset, offsetStr)
		}
		if cfg.MaxOffset > 0 && offset > cfg.MaxOffset {
			return OffsetRequest{}, fmt.Errorf("%w: %d exceeds MaxOffset %d", ErrInvalidOffset, offset, cfg.MaxOffset)
		}
		if offset%limit != 0 {
			return OffsetRequest{}, fmt.Errorf("%w: %d must be a multiple of limit %d (use %q instead)",
				ErrInvalidOffset, offset, limit, cfg.PageParam)
		}
		return OffsetRequest{Page: offset/limit + 1, Limit: limit}, nil

	default:
		return OffsetRequest{Page: 1, Limit: limit}, nil
	}
}

func parseLimit(raw string, cfg Config) (int, error) {
	if raw == "" {
		return cfg.DefaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: %q is not an integer", ErrInvalidLimit, raw)
	}
	if n < cfg.MinLimit {
		return 0, fmt.Errorf("%w: %d is below MinLimit %d", ErrInvalidLimit, n, cfg.MinLimit)
	}
	if n > cfg.MaxLimit {
		return 0, fmt.Errorf("%w: %d exceeds MaxLimit %d", ErrInvalidLimit, n, cfg.MaxLimit)
	}
	return n, nil
}
