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

// Command is a command-style use case: it performs the action described by the
// input and reports only success or failure. Use it for writes/mutations that
// don't return a value (CQRS "command").
//
// I is the input type. Use a single struct for I so the signature stays stable
// as the use case grows new parameters.
type Command[I any] interface {
	Execute(ctx context.Context, input I) error
}

// Query is a use case that returns a result of type O along with an error — for
// example, a read, or a command that yields a value such as a generated
// identifier (CQRS "query").
type Query[I, O any] interface {
	Execute(ctx context.Context, input I) (O, error)
}

// CommandFunc adapts an ordinary function to the Command interface, mirroring
// the http.HandlerFunc pattern.
type CommandFunc[I any] func(ctx context.Context, input I) error

// Execute calls f(ctx, input).
func (f CommandFunc[I]) Execute(ctx context.Context, input I) error {
	return f(ctx, input)
}

// QueryFunc adapts an ordinary function to the Query interface.
type QueryFunc[I, O any] func(ctx context.Context, input I) (O, error)

// Execute calls f(ctx, input).
func (f QueryFunc[I, O]) Execute(ctx context.Context, input I) (O, error) {
	return f(ctx, input)
}
