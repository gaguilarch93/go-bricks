package pagination_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gaguilarch93/go-bricks/pagination"
)

type user struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ExampleParseOffset shows a net/http handler using offset pagination with
// HATEOAS links and an RFC 8288 Link header.
func ExampleParseOffset() {
	cfg := pagination.DefaultConfig()

	handler := func(w http.ResponseWriter, r *http.Request) {
		req, err := pagination.ParseOffset(r.URL.Query(), cfg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		users := []user{{ID: 1, Name: "ada"}, {ID: 2, Name: "linus"}}
		var total int64 = 42

		page := pagination.NewOffsetPage(users, req, total)
		links := pagination.NewLinkBuilder(r, cfg).Offset(req, page.Meta)
		page = page.WithLinks(links)

		w.Header().Set("Link", links.Header())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "?page=1&limit=20")
	var out pagination.OffsetPage[user]
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()

	fmt.Println(out.Meta.Page, out.Meta.Size, *out.Meta.Total, out.Meta.HasMore)
	// Output: 1 20 42 true
}

// ExampleCursorCodec shows signed cursor encoding/decoding.
func ExampleCursorCodec() {
	codec := pagination.CursorCodec{Secret: []byte("change-me-in-prod")}
	enc, _ := codec.Encode(pagination.CursorPayload{"id": "u_100"})
	got, _ := codec.Decode(enc)
	fmt.Println(got["id"])
	// Output: u_100
}
