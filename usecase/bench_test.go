package usecase_test

import (
	"context"
	"testing"

	"github.com/gaguilarch93/go-bricks/usecase"
)

type benchInput struct {
	ok bool
}

func (in benchInput) Validate(ctx context.Context) error {
	if !in.ok {
		return errBoomBench
	}
	return nil
}

type benchPlain struct {
	n int
}

var errBoomBench = context.Canceled // any non-nil error reused to avoid allocs

func BenchmarkRun_NoValidator(b *testing.B) {
	ctx := context.Background()
	uc := usecase.Func[benchPlain](func(ctx context.Context, in benchPlain) error { return nil })
	in := benchPlain{n: 1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = usecase.Run[benchPlain](ctx, uc, in)
	}
}

func BenchmarkRun_WithValidator(b *testing.B) {
	ctx := context.Background()
	uc := usecase.Func[benchInput](func(ctx context.Context, in benchInput) error { return nil })
	in := benchInput{ok: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = usecase.Run[benchInput](ctx, uc, in)
	}
}

func BenchmarkRunResult_WithValidator(b *testing.B) {
	ctx := context.Background()
	uc := usecase.ResultFunc[benchInput, int](func(ctx context.Context, in benchInput) (int, error) { return 1, nil })
	in := benchInput{ok: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = usecase.RunResult[benchInput, int](ctx, uc, in)
	}
}
