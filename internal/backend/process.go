package backend

import (
	"context"
	"errors"
	"myenv/internal/runner"
)

type processObserverKey struct{}

// ProcessObserver durably registers a child before launch and receives its
// runner result after execution. Begin may assign process supervision fields.
type ProcessObserver func(context.Context, *runner.Process) (func(error) error, error)

func WithProcessObserver(ctx context.Context, observer ProcessObserver) context.Context {
	return context.WithValue(ctx, processObserverKey{}, observer)
}

// Backend operations expose cancellation as an error after runner has waited
// for the child tree. User commands keep runner's native exit-code contract.
func executeBackend(ctx context.Context, p runner.Process) (int, error) {
	var finish func(error) error
	if observer, ok := ctx.Value(processObserverKey{}).(ProcessObserver); ok {
		var err error
		finish, err = observer(ctx, &p)
		if err != nil {
			return 1, err
		}
	}
	code, err := runner.Execute(ctx, p)
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	if finish != nil {
		err = errors.Join(err, finish(err))
	}
	return code, err
}
