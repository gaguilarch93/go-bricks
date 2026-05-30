package pagination

import (
	"errors"
	"net/http"
)

// Sentinel errors. All parsing errors wrap one of these so callers can use
// errors.Is to branch (e.g. when mapping to HTTP status codes).
var (
	ErrInvalidLimit  = errors.New("pagination: invalid limit")
	ErrInvalidPage   = errors.New("pagination: invalid page")
	ErrInvalidOffset = errors.New("pagination: invalid offset")
	ErrInvalidCursor = errors.New("pagination: invalid cursor")
	ErrCursorExpired = errors.New("pagination: cursor expired")
	ErrCursorTamper  = errors.New("pagination: cursor signature mismatch")
	ErrCursorConfig  = errors.New("pagination: invalid cursor codec configuration")
)

// MapErrorToHTTPStatus is a convenience that maps pagination sentinel errors
// to standard HTTP status codes. Unknown errors map to 500.
//
//	if err != nil {
//	    http.Error(w, err.Error(), pagination.MapErrorToHTTPStatus(err))
//	    return
//	}
func MapErrorToHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrInvalidPage),
		errors.Is(err, ErrInvalidOffset),
		errors.Is(err, ErrInvalidCursor),
		errors.Is(err, ErrCursorExpired),
		errors.Is(err, ErrCursorTamper):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
