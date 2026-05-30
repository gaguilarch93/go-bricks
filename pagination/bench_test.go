package pagination_test

import (
	"net/url"
	"testing"

	"github.com/gaguilarch93/go-bricks/pagination"
)

func BenchmarkParseOffset(b *testing.B) {
	cfg := pagination.DefaultConfig()
	v := url.Values{"page": {"7"}, "limit": {"50"}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pagination.ParseOffset(v, cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseCursor(b *testing.B) {
	cfg := pagination.DefaultConfig()
	v := url.Values{"cursor": {"abc"}, "limit": {"50"}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := pagination.ParseCursor(v, cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCursorEncode_Unsigned(b *testing.B) {
	c := pagination.CursorCodec{}
	payload := pagination.CursorPayload{"created_at": "2025-01-01T00:00:00Z", "id": 12345}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := c.Encode(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCursorEncode_Signed(b *testing.B) {
	c := pagination.CursorCodec{Secret: []byte("s3cret")}
	payload := pagination.CursorPayload{"created_at": "2025-01-01T00:00:00Z", "id": 12345}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := c.Encode(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCursorRoundTrip_Signed(b *testing.B) {
	c := pagination.CursorCodec{Secret: []byte("s3cret")}
	enc, _ := c.Encode(pagination.CursorPayload{"id": 12345})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := c.Decode(enc); err != nil {
			b.Fatal(err)
		}
	}
}
