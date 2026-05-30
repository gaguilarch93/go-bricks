package pagination

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// CursorRequest is a validated cursor-pagination request.
type CursorRequest struct {
	Cursor string
	Limit  int
}

// ParseCursor reads and validates a CursorRequest from query parameters.
// The cursor is returned opaquely; decode it with a CursorCodec to recover
// the payload your repository needs.
func ParseCursor(values url.Values, cfg Config) (CursorRequest, error) {
	cfg = cfg.withDefaults()
	limit, err := parseLimit(values.Get(cfg.LimitParam), cfg)
	if err != nil {
		return CursorRequest{}, err
	}
	return CursorRequest{
		Cursor: values.Get(cfg.CursorParam),
		Limit:  limit,
	}, nil
}

// CursorPayload is the data you stash inside an opaque cursor. Keep it small
// and serializable: typically a sort key and a tiebreaker (e.g. created_at +
// id). Avoid putting filters or user-controlled state here.
//
// Number handling: JSON has no integer type, and Go's encoding/json normally
// decodes numbers into float64 — which corrupts IDs above 2^53. To preserve
// precision, CursorCodec.Decode uses a json.Decoder with UseNumber(), so
// numeric fields arrive as json.Number. Use .Int64() / .Float64() / .String()
// to convert.
type CursorPayload map[string]any

// DefaultMaxCursorBytes bounds the length of an encoded cursor accepted by
// Decode when CursorCodec.MaxCursorBytes is left at zero. It prevents a client
// from forcing large base64/JSON allocations with an oversized token. 8 KiB is
// comfortably larger than any sane cursor yet far below typical HTTP URL limits.
const DefaultMaxCursorBytes = 8192

// CursorCodec encodes and decodes opaque cursors. A zero-value codec produces
// unsigned cursors (safe for trusted clients).
//
//   - Set Secret to enable HMAC-SHA256 signing (strongly recommended for any
//     public API to prevent tampering and forgery). Secret is the active key:
//     it both signs (Encode) and verifies (Decode).
//   - Set RetiredSecrets to support zero-downtime key rotation. They are used
//     for verification ONLY — never to sign. On rotation, move the old Secret
//     into RetiredSecrets and set a fresh Secret; cursors issued under the old
//     key keep decoding until they expire or you drop the retired key. Requires
//     Secret to be set.
//   - Set TTL to expire cursors after a duration. TTL REQUIRES a Secret —
//     otherwise the timestamp is forgeable and the expiry is meaningless.
//     Encode and Decode return ErrCursorConfig if TTL is set without Secret.
//   - Set MaxCursorBytes to override DefaultMaxCursorBytes for the maximum
//     encoded cursor length accepted by Decode.
type CursorCodec struct {
	Secret         []byte
	RetiredSecrets [][]byte
	TTL            time.Duration
	MaxCursorBytes int
}

type envelope struct {
	V int           `json:"v"`
	T int64         `json:"t,omitempty"` // unix milliseconds when set
	P CursorPayload `json:"p"`
}

func (c CursorCodec) validate() error {
	if c.TTL > 0 && len(c.Secret) == 0 {
		return fmt.Errorf("%w: TTL requires Secret (unsigned timestamps can be forged)", ErrCursorConfig)
	}
	if len(c.RetiredSecrets) > 0 && len(c.Secret) == 0 {
		return fmt.Errorf("%w: RetiredSecrets requires an active Secret", ErrCursorConfig)
	}
	return nil
}

// verifySig reports whether sig is a valid HMAC of body under the active Secret
// or any RetiredSecret. Each candidate is compared in constant time.
func (c CursorCodec) verifySig(body string, sig []byte) bool {
	if macEqual(c.Secret, body, sig) {
		return true
	}
	for _, k := range c.RetiredSecrets {
		if len(k) == 0 {
			continue
		}
		if macEqual(k, body, sig) {
			return true
		}
	}
	return false
}

func macEqual(key []byte, body string, sig []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	return hmac.Equal(sig, mac.Sum(nil))
}

// Encode serializes payload into a URL-safe opaque string. When Secret is set,
// the payload is HMAC-SHA256 signed; the signature is appended after a dot.
func (c CursorCodec) Encode(payload CursorPayload) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}
	env := envelope{V: 1, P: payload}
	if len(c.Secret) > 0 || c.TTL > 0 {
		env.T = time.Now().UnixMilli()
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("pagination: encode cursor: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	if len(c.Secret) == 0 {
		return body, nil
	}
	mac := hmac.New(sha256.New, c.Secret)
	mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return body + "." + sig, nil
}

// Decode reverses Encode. It verifies the signature when Secret is set and
// enforces TTL when configured. Returns ErrInvalidCursor, ErrCursorTamper,
// ErrCursorExpired, or ErrCursorConfig on failure.
func (c CursorCodec) Decode(s string) (CursorPayload, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if s == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidCursor)
	}
	maxLen := c.MaxCursorBytes
	if maxLen <= 0 {
		maxLen = DefaultMaxCursorBytes
	}
	if len(s) > maxLen {
		return nil, fmt.Errorf("%w: encoded length %d exceeds max %d", ErrInvalidCursor, len(s), maxLen)
	}
	body := s
	if len(c.Secret) > 0 {
		dot := strings.LastIndexByte(s, '.')
		if dot < 0 {
			return nil, fmt.Errorf("%w: missing signature", ErrCursorTamper)
		}
		body = s[:dot]
		gotSig, err := base64.RawURLEncoding.DecodeString(s[dot+1:])
		if err != nil {
			return nil, fmt.Errorf("%w: signature not base64", ErrCursorTamper)
		}
		if !c.verifySig(body, gotSig) {
			return nil, ErrCursorTamper
		}
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("%w: body not base64", ErrInvalidCursor)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // preserve int64 precision in CursorPayload
	var env envelope
	if err := dec.Decode(&env); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}
	if env.V != 1 {
		return nil, fmt.Errorf("%w: unsupported version %d", ErrInvalidCursor, env.V)
	}
	if c.TTL > 0 {
		// Encode always writes T when TTL is set, so a missing/zero
		// timestamp on Decode means either (a) the cursor predates the
		// TTL policy or (b) it was crafted. Either way, reject it —
		// silently accepting "pre-TTL" cursors would let stale tokens
		// live indefinitely after a policy rotation.
		if env.T <= 0 {
			return nil, ErrCursorExpired
		}
		issued := time.UnixMilli(env.T)
		if time.Since(issued) > c.TTL {
			return nil, ErrCursorExpired
		}
	}
	if env.P == nil {
		// Normalize: callers can safely read or write to the returned map
		// even when the original payload was nil/null.
		env.P = CursorPayload{}
	}
	return env.P, nil
}
