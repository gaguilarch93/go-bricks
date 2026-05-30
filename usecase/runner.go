package usecase

import (
	"context"
	"fmt"
)

// Validate runs optional input validation. If input implements Validator, its
// Validate method is invoked and any non-nil error is wrapped with
// ErrValidation. Inputs that do not implement Validator are always considered
// valid and Validate returns nil.
func Validate(ctx context.Context, input any) error {
	v, ok := input.(Validator)
	if !ok {
		return nil
	}
	if err := v.Validate(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrValidation, err)
	}
	return nil
}

// Run validates input (when it implements Validator) and then executes uc.
//
// It returns:
//   - ErrNilUseCase if uc is nil (a nil interface, or a nil CommandFunc);
//   - an error wrapping ErrValidation if input validation fails (Execute is
//     not called);
//   - otherwise whatever uc.Execute returns.
//
// Caveat: a non-nil interface that wraps a nil pointer to your own type is not
// detected here; calling its Execute may panic. Prefer value receivers or
// non-nil pointers for use case implementations.
func Run[I any](ctx context.Context, uc Command[I], input I) error {
	if uc == nil {
		return ErrNilUseCase
	}
	if err := Validate(ctx, input); err != nil {
		return err
	}
	return uc.Execute(ctx, input)
}

// RunResult validates input (when it implements Validator) and then executes
// uc, returning its result.
//
// On a nil use case (a nil interface or a nil QueryFunc) or a validation
// failure it returns the zero value of O together with the error (ErrNilUseCase
// or an ErrValidation-wrapped cause). The typed-nil-pointer caveat described on
// Run applies here too.
func RunResult[I, O any](ctx context.Context, uc Query[I, O], input I) (O, error) {
	if uc == nil {
		var zero O
		return zero, ErrNilUseCase
	}
	if err := Validate(ctx, input); err != nil {
		var zero O
		return zero, err
	}
	return uc.Execute(ctx, input)
}
