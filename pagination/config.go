package pagination

import "fmt"

// Config controls parsing defaults and safety bounds.
//
// A zero-value Config is usable: any unset numeric or string field is
// transparently filled with its DefaultConfig() counterpart at parse time.
// Calling Validate() at startup is still recommended to catch misconfigured
// combinations (e.g. DefaultLimit > MaxLimit) before serving traffic.
//
// MaxOffset semantics:
//
//   - > 0   : reject offsets above this value
//   - == 0  : use DefaultConfig().MaxOffset (10_000)
//   - < 0   : disable the offset cap (NOT recommended for public APIs)
type Config struct {
	DefaultLimit int
	MinLimit     int
	MaxLimit     int
	MaxOffset    int

	LimitParam  string
	PageParam   string
	OffsetParam string
	CursorParam string
}

// DefaultConfig returns a safe, opinionated baseline: limit defaults to 20,
// capped at 100, offset capped at 10_000.
func DefaultConfig() Config {
	return Config{
		DefaultLimit: 20,
		MinLimit:     1,
		MaxLimit:     100,
		MaxOffset:    10_000,
		LimitParam:   "limit",
		PageParam:    "page",
		OffsetParam:  "offset",
		CursorParam:  "cursor",
	}
}

// Validate returns an error if the configuration is internally inconsistent.
// Call it once at startup; do not call per request.
//
// Note: withDefaults() runs first, so MinLimit/MaxLimit/DefaultLimit are
// guaranteed >= 1 here. The defensive checks below cover future-proofing in
// case those defaults change.
func (c Config) Validate() error {
	c = c.withDefaults()
	if c.MinLimit < 1 {
		return fmt.Errorf("pagination: MinLimit must be >= 1, got %d", c.MinLimit)
	}
	if c.MaxLimit < c.MinLimit {
		return fmt.Errorf("pagination: MaxLimit (%d) must be >= MinLimit (%d)", c.MaxLimit, c.MinLimit)
	}
	if c.DefaultLimit < c.MinLimit || c.DefaultLimit > c.MaxLimit {
		return fmt.Errorf("pagination: DefaultLimit (%d) must be within [%d, %d]", c.DefaultLimit, c.MinLimit, c.MaxLimit)
	}
	return nil
}

// withDefaults fills any unset field from DefaultConfig. Negative MaxOffset is
// preserved (it disables the cap). Zero MaxOffset means "use default".
func (c Config) withDefaults() Config {
	d := DefaultConfig()
	// Track whether DefaultLimit was user-supplied so that explicit
	// misconfigurations (e.g. user sets DefaultLimit=200 with MaxLimit=100)
	// are preserved verbatim and surfaced by Validate(), instead of being
	// silently corrected here.
	defaultLimitWasSet := c.DefaultLimit > 0

	if c.DefaultLimit <= 0 {
		c.DefaultLimit = d.DefaultLimit
	}
	if c.MinLimit <= 0 {
		c.MinLimit = d.MinLimit
	}
	if c.MaxLimit <= 0 {
		c.MaxLimit = d.MaxLimit
	}
	if c.MaxOffset == 0 {
		c.MaxOffset = d.MaxOffset
	}
	// Only clamp DefaultLimit when we filled it ourselves; preserves the
	// user's explicit value so Validate() can complain.
	if !defaultLimitWasSet {
		if c.DefaultLimit < c.MinLimit {
			c.DefaultLimit = c.MinLimit
		}
		if c.DefaultLimit > c.MaxLimit {
			c.DefaultLimit = c.MaxLimit
		}
	}
	if c.LimitParam == "" {
		c.LimitParam = d.LimitParam
	}
	if c.PageParam == "" {
		c.PageParam = d.PageParam
	}
	if c.OffsetParam == "" {
		c.OffsetParam = d.OffsetParam
	}
	if c.CursorParam == "" {
		c.CursorParam = d.CursorParam
	}
	return c
}
