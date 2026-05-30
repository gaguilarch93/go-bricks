package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gaguilarch93/go-bricks/usecase"
)

// --- test fixtures ---

type noValidateInput struct {
	v int
}

type valueValidateInput struct {
	ok bool
}

func (in valueValidateInput) Validate(ctx context.Context) error {
	if !in.ok {
		return errors.New("not ok")
	}
	return nil
}

type ptrValidateInput struct {
	ok bool
}

func (in *ptrValidateInput) Validate(ctx context.Context) error {
	if !in.ok {
		return errors.New("not ok")
	}
	return nil
}

var errBoom = errors.New("boom")

// --- UseCase / Run ---

func TestRun_NoValidatorExecutes(t *testing.T) {
	called := false
	uc := usecase.Func[noValidateInput](func(ctx context.Context, in noValidateInput) error {
		called = true
		if in.v != 7 {
			t.Fatalf("input not threaded: %+v", in)
		}
		return nil
	})
	if err := usecase.Run[noValidateInput](context.Background(), uc, noValidateInput{v: 7}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !called {
		t.Fatal("execute was not called")
	}
}

func TestRun_ValidationPassesThenExecutes(t *testing.T) {
	called := false
	uc := usecase.Func[valueValidateInput](func(ctx context.Context, in valueValidateInput) error {
		called = true
		return nil
	})
	if err := usecase.Run[valueValidateInput](context.Background(), uc, valueValidateInput{ok: true}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !called {
		t.Fatal("execute should run after successful validation")
	}
}

func TestRun_ValidationFailsShortCircuits(t *testing.T) {
	uc := usecase.Func[valueValidateInput](func(ctx context.Context, in valueValidateInput) error {
		t.Fatal("execute must not run when validation fails")
		return nil
	})
	err := usecase.Run[valueValidateInput](context.Background(), uc, valueValidateInput{ok: false})
	if !errors.Is(err, usecase.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestRun_PropagatesExecuteError(t *testing.T) {
	uc := usecase.Func[noValidateInput](func(ctx context.Context, in noValidateInput) error {
		return errBoom
	})
	err := usecase.Run[noValidateInput](context.Background(), uc, noValidateInput{})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected errBoom, got %v", err)
	}
	if errors.Is(err, usecase.ErrValidation) {
		t.Fatal("execute error must not be reported as a validation error")
	}
}

func TestRun_NilUseCase(t *testing.T) {
	var uc usecase.UseCase[noValidateInput]
	if err := usecase.Run[noValidateInput](context.Background(), uc, noValidateInput{}); !errors.Is(err, usecase.ErrNilUseCase) {
		t.Fatalf("expected ErrNilUseCase, got %v", err)
	}
}

// --- ResultUseCase / RunResult ---

func TestRunResult_ReturnsValue(t *testing.T) {
	uc := usecase.ResultFunc[valueValidateInput, string](func(ctx context.Context, in valueValidateInput) (string, error) {
		return "done", nil
	})
	out, err := usecase.RunResult[valueValidateInput, string](context.Background(), uc, valueValidateInput{ok: true})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out != "done" {
		t.Fatalf("got %q", out)
	}
}

func TestRunResult_ValidationFailsReturnsZero(t *testing.T) {
	uc := usecase.ResultFunc[valueValidateInput, string](func(ctx context.Context, in valueValidateInput) (string, error) {
		t.Fatal("execute must not run when validation fails")
		return "x", nil
	})
	out, err := usecase.RunResult[valueValidateInput, string](context.Background(), uc, valueValidateInput{ok: false})
	if !errors.Is(err, usecase.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if out != "" {
		t.Fatalf("expected zero value, got %q", out)
	}
}

func TestRunResult_NilUseCase(t *testing.T) {
	var uc usecase.ResultUseCase[noValidateInput, int]
	out, err := usecase.RunResult[noValidateInput, int](context.Background(), uc, noValidateInput{})
	if !errors.Is(err, usecase.ErrNilUseCase) {
		t.Fatalf("expected ErrNilUseCase, got %v", err)
	}
	if out != 0 {
		t.Fatalf("expected zero value, got %d", out)
	}
}

// --- Validate helper + method-set behavior ---

func TestValidate_NonValidatorIsAlwaysValid(t *testing.T) {
	if err := usecase.Validate(context.Background(), noValidateInput{}); err != nil {
		t.Fatalf("non-validator input should be valid, got %v", err)
	}
}

func TestValidate_WrapsCause(t *testing.T) {
	err := usecase.Validate(context.Background(), valueValidateInput{ok: false})
	if !errors.Is(err, usecase.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	// The underlying cause must remain inspectable.
	if err.Error() == usecase.ErrValidation.Error() {
		t.Fatal("expected cause to be wrapped alongside the sentinel")
	}
}

func TestValidate_PointerReceiverMethodSet(t *testing.T) {
	// Value of a type whose Validate has a pointer receiver does NOT satisfy
	// Validator, so validation is skipped (documented behavior).
	if err := usecase.Validate(context.Background(), ptrValidateInput{ok: false}); err != nil {
		t.Fatalf("value with pointer-receiver Validate should skip validation, got %v", err)
	}
	// The pointer does satisfy Validator and is validated.
	if err := usecase.Validate(context.Background(), &ptrValidateInput{ok: false}); !errors.Is(err, usecase.ErrValidation) {
		t.Fatalf("pointer input should be validated, got %v", err)
	}
}

func TestRun_PointerInputIsValidated(t *testing.T) {
	uc := usecase.Func[*ptrValidateInput](func(ctx context.Context, in *ptrValidateInput) error {
		t.Fatal("execute must not run when pointer validation fails")
		return nil
	})
	err := usecase.Run[*ptrValidateInput](context.Background(), uc, &ptrValidateInput{ok: false})
	if !errors.Is(err, usecase.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestValidate_ReceivesContext(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "v")
	in := ctxAwareInput{key: ctxKey{}, want: "v"}
	if err := usecase.Validate(ctx, in); err != nil {
		t.Fatalf("validator should see context value: %v", err)
	}
}

type ctxAwareInput struct {
	key  any
	want string
}

func (in ctxAwareInput) Validate(ctx context.Context) error {
	if ctx.Value(in.key) != in.want {
		return errors.New("missing context value")
	}
	return nil
}
