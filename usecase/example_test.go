package usecase_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/gaguilarch93/go-bricks/usecase"
)

// createUserInput validates itself: its Validate method runs automatically
// before the use case executes.
type createUserInput struct {
	Email string
}

func (in createUserInput) Validate(ctx context.Context) error {
	if in.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

// createUser is a Query that returns the new user's ID.
type createUser struct{}

func (createUser) Execute(ctx context.Context, in createUserInput) (string, error) {
	return "user:" + in.Email, nil
}

func ExampleRunResult() {
	ctx := context.Background()

	id, err := usecase.RunResult[createUserInput, string](ctx, createUser{}, createUserInput{Email: "a@b.com"})
	fmt.Println(id, err)

	// Invalid input never reaches Execute.
	_, err = usecase.RunResult[createUserInput, string](ctx, createUser{}, createUserInput{})
	fmt.Println(errors.Is(err, usecase.ErrValidation))

	// Output:
	// user:a@b.com <nil>
	// true
}

// deleteUser is a Command (no return value).
type deleteUser struct{}

func (deleteUser) Execute(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("not found")
	}
	return nil
}

func ExampleRun() {
	err := usecase.Run[string](context.Background(), deleteUser{}, "user:42")
	fmt.Println(err)
	// Output:
	// <nil>
}
