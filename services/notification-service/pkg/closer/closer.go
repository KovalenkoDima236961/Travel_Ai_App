package closer

import (
	"context"
	"fmt"
	"sync"
)

type closeFunc struct {
	name string
	fn   func(context.Context) error
}

var (
	mu      sync.Mutex
	closers []closeFunc
)

func Add(name string, fn func(context.Context) error) {
	if fn == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	closers = append(closers, closeFunc{name: name, fn: fn})
}

func CloseAll(ctx context.Context) error {
	mu.Lock()
	pending := make([]closeFunc, len(closers))
	copy(pending, closers)
	closers = nil
	mu.Unlock()

	var firstErr error
	for i := len(pending) - 1; i >= 0; i-- {
		if err := pending[i].fn(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close %s: %w", pending[i].name, err)
		}
	}

	return firstErr
}
