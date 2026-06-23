package closer

import (
	"context"
	"errors"
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

// Add registers a named shutdown function. Resources are closed in LIFO order.
func Add(name string, fn func(context.Context) error) {
	mu.Lock()
	defer mu.Unlock()
	closers = append(closers, closeFunc{name: name, fn: fn})
}

// CloseAll closes all registered resources in LIFO order.
func CloseAll(ctx context.Context) error {
	mu.Lock()
	items := make([]closeFunc, len(closers))
	copy(items, closers)
	closers = nil
	mu.Unlock()

	var result error
	for i := len(items) - 1; i >= 0; i-- {
		if err := items[i].fn(ctx); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}
