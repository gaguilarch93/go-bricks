// Package usecase defines small, composable interfaces for application use
// cases (interactors) in a clean / hexagonal architecture.
//
// It provides two generic interfaces:
//
//   - Command[I]    — a command: input in, error out.
//   - Query[I, O]   — a read, or a command that yields a value:
//     input in, (result, error) out.
//
// Inputs may OPTIONALLY implement Validator to validate themselves. The Run and
// RunResult helpers invoke Validate (when the input implements it) before
// executing the use case, so business logic never runs on invalid input.
// Inputs that do not implement Validator are executed as-is.
//
// The package has no third-party dependencies and imposes no framework: the
// interfaces are intentionally tiny so they compose with your own transport,
// logging, and transaction layers.
//
// Typical wiring:
//
//	type CreateUser struct{ repo UserRepo }
//
//	type CreateUserInput struct{ Email string }
//
//	func (in CreateUserInput) Validate(ctx context.Context) error {
//	    if in.Email == "" {
//	        return errors.New("email is required")
//	    }
//	    return nil
//	}
//
//	func (uc CreateUser) Execute(ctx context.Context, in CreateUserInput) (string, error) {
//	    return uc.repo.Insert(ctx, in.Email)
//	}
//
//	// CreateUserInput.Validate runs automatically before Execute.
//	id, err := usecase.RunResult[CreateUserInput, string](ctx, CreateUser{repo}, in)
package usecase
