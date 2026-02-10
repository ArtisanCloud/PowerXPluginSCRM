package driver

import "context"

type progressReporter func(current, total int, stage string)

type progressKey struct{}

func WithProgressReporter(ctx context.Context, reporter func(current, total int, stage string)) context.Context {
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, progressKey{}, progressReporter(reporter))
}

func progressFromContext(ctx context.Context) progressReporter {
	if ctx == nil {
		return nil
	}
	if v := ctx.Value(progressKey{}); v != nil {
		if reporter, ok := v.(progressReporter); ok {
			return reporter
		}
	}
	return nil
}
