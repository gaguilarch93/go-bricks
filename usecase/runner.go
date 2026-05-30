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
//   - ErrNilUseCase if uc is a nil interface value;
//   - an error wrapping ErrValidation if input validation fails (Execute is
//     not called);
//   - otherwise whatever uc.Execute returns.
func Run[I any](ctx context.Context, uc UseCase[I], input I) error {
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
// On a nil use case or a validation failure it returns the zero value of O
// together with the error (ErrNilUseCase or an ErrValidation-wrapped cause).
func RunResult[I, O any](ctx context.Context, uc ResultUseCase[I, O], input I) (O, error) {
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
