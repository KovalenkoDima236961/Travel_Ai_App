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

// Stack closes registered resources in last-in-first-out order.
type Stack struct {
	mu      sync.Mutex
	closers []closeFunc
}

// New creates an empty shutdown stack.
func New() *Stack {
	return &Stack{}
}

// Add registers a named shutdown function. Resources are closed in LIFO order.
func (s *Stack) Add(name string, fn func(context.Context) error) {
	if fn == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.closers = append(s.closers, closeFunc{name: name, fn: fn})
}

// CloseAll closes all registered resources in LIFO order.
func (s *Stack) CloseAll(ctx context.Context) error {
	s.mu.Lock()
	items := make([]closeFunc, len(s.closers))
	copy(items, s.closers)
	s.closers = nil
	s.mu.Unlock()

	var result error
	for i := len(items) - 1; i >= 0; i-- {
		if err := items[i].fn(ctx); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}
