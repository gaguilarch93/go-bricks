package usecase

import "errors"

// Sentinel errors returned by this package. Use errors.Is to detect them.
var (
	// ErrValidation wraps any error returned by an input's Validate method.
	// Run, RunResult and Validate return it (wrapping the original cause) when
	// input validation fails, so callers can branch on validation errors with
	// errors.Is(err, ErrValidation) while still inspecting the cause via
	// errors.As / errors.Unwrap.
	ErrValidation = errors.New("usecase: input validation failed")

	// ErrNilUseCase is returned by Run and RunResult when the supplied use case
	// is a nil interface value.
	ErrNilUseCase = errors.New("usecase: nil use case")
)
