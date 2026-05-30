package usecase

import "context"

// Validator is an OPTIONAL interface that a use case input may implement to
// validate itself. When an input implements Validator, the Run and RunResult
// helpers (and the standalone Validate function) call Validate before the use
// case executes and short-circuit on a non-nil error.
//
// Method-set note: if Validate is declared on a pointer receiver (*T), only a
// *T value satisfies Validator. Pass the pointer as the input in that case, or
// declare Validate on a value receiver so both T and *T satisfy it.
type Validator interface {
	Validate(ctx context.Context) error
}

// UseCase is a command-style use case: it performs the action described by the
// input and reports only success or failure.
//
// I is the input type. Use a single struct for I so the signature stays stable
// as the use case grows new parameters.
type UseCase[I any] interface {
	Execute(ctx context.Context, input I) error
}

// ResultUseCase is a use case that returns a result of type O along with an
// error — for example, a query, or a command that yields a value such as a
// generated identifier.
type ResultUseCase[I, O any] interface {
	Execute(ctx context.Context, input I) (O, error)
}

// Func adapts an ordinary function to the UseCase interface, mirroring the
// http.HandlerFunc pattern.
type Func[I any] func(ctx context.Context, input I) error

// Execute calls f(ctx, input).
func (f Func[I]) Execute(ctx context.Context, input I) error {
	return f(ctx, input)
}

// ResultFunc adapts an ordinary function to the ResultUseCase interface.
type ResultFunc[I, O any] func(ctx context.Context, input I) (O, error)

// Execute calls f(ctx, input).
func (f ResultFunc[I, O]) Execute(ctx context.Context, input I) (O, error) {
	return f(ctx, input)
}
