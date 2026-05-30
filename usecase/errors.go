package usecase

import "errors"

// Sentinel errors returned by this package. Use errors.Is to detect them.
var (
	// ErrValidation wraps any error returned by an input's Validate method.
	// Run, RunResult and Validate return it (joined with the original cause via
	// fmt.Errorf's multi-%w) when input validation fails, so callers can branch
	// with errors.Is(err, ErrValidation) and recover the cause with errors.As.
	//
	// Note: because the returned error wraps two errors, the single-step
	// errors.Unwrap function returns nil for it — use errors.Is / errors.As,
	// which traverse the full error tree.
	ErrValidation = errors.New("usecase: input validation failed")

	// ErrNilUseCase is returned by Run and RunResult when the supplied use case
	// is nil — either a nil interface, or a nil CommandFunc/QueryFunc adapter.
	ErrNilUseCase = errors.New("usecase: nil use case")
)
