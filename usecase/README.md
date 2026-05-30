# usecase

Tiny, dependency-free interfaces for application **use cases** (interactors) in
a clean / hexagonal architecture, with **optional, automatic input validation**.

## Why

- **Two shapes, no boilerplate**: a command (`UseCase[I]`) and a value-returning
  use case (`ResultUseCase[I, O]`).
- **Validation that's opt-in per input**: an input that implements `Validator`
  is validated automatically before the use case runs — others just run.
- **Composable**: the interfaces are intentionally minimal, so they layer
  cleanly under your transport, logging, and transaction code.
- **Zero third-party dependencies.**

## Install

```bash
go get github.com/gaguilarch93/go-bricks/usecase
```

## Interfaces

```go
// Command: input in, error out.
type UseCase[I any] interface {
    Execute(ctx context.Context, input I) error
}

// Query / value-returning command: input in, (result, error) out.
type ResultUseCase[I, O any] interface {
    Execute(ctx context.Context, input I) (O, error)
}

// Optional — implement on an input to have it validated automatically.
type Validator interface {
    Validate(ctx context.Context) error
}
```

## Usage

```go
type CreateUserInput struct {
    Email string
}

// Optional: implementing Validator makes Run/RunResult validate first.
func (in CreateUserInput) Validate(ctx context.Context) error {
    if in.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

type CreateUser struct {
    repo UserRepo
}

func (uc CreateUser) Execute(ctx context.Context, in CreateUserInput) (string, error) {
    return uc.repo.Insert(ctx, in.Email)
}

// Run validates the input (because it implements Validator), then executes.
id, err := usecase.RunResult[CreateUserInput, string](ctx, CreateUser{repo}, in)
switch {
case errors.Is(err, usecase.ErrValidation):
    // 400 — input failed validation; inspect the cause with errors.Unwrap/As.
case err != nil:
    // 500 — execution error.
default:
    // ok — use id.
}
```

For a command without a result, use `Run`:

```go
err := usecase.Run[string](ctx, DeleteUser{repo}, userID)
```

### Function adapters

Wrap a plain function as a use case (like `http.HandlerFunc`):

```go
uc := usecase.Func[CreateUserInput](func(ctx context.Context, in CreateUserInput) error {
    return nil
})
```

`usecase.ResultFunc[I, O]` does the same for value-returning use cases.

## Validation semantics

- `Run` / `RunResult` call `Validate` **only** when the input implements
  `Validator`; otherwise the input is considered valid.
- A validation failure is wrapped as `ErrValidation` (the original cause stays
  reachable via `errors.Unwrap` / `errors.As`); `Execute` is **not** called.
- A nil use case returns `ErrNilUseCase`.
- You can validate without running via `usecase.Validate(ctx, input)`.

### Method-set gotcha

If `Validate` is declared on a **pointer** receiver (`*T`), only a `*T`
satisfies `Validator`. Pass the pointer as the input, or declare `Validate` on a
value receiver so both `T` and `*T` qualify.

## Errors

| Sentinel         | Meaning                                  | Typical HTTP |
|------------------|------------------------------------------|--------------|
| `ErrValidation`  | input `Validate` returned an error       | 400          |
| `ErrNilUseCase`  | a nil use case was passed to Run/RunResult | 500        |

Use `errors.Is` to branch.
