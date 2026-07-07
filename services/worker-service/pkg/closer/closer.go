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

// Stack closes registered resources in last-in-first-out order.
type Stack struct {
	mu      sync.Mutex
	closers []closeFunc
}

// New creates an empty shutdown stack.
func New() *Stack {
	return &Stack{}
}

// Add registers a resource shutdown callback.
func (s *Stack) Add(name string, fn func(context.Context) error) {
	if fn == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.closers = append(s.closers, closeFunc{name: name, fn: fn})
}

// CloseAll closes all registered resources and returns the first close error.
func (s *Stack) CloseAll(ctx context.Context) error {
	s.mu.Lock()
	pending := make([]closeFunc, len(s.closers))
	copy(pending, s.closers)
	s.closers = nil
	s.mu.Unlock()

	var firstErr error
	for i := len(pending) - 1; i >= 0; i-- {
		if err := pending[i].fn(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close %s: %w", pending[i].name, err)
		}
	}

	return firstErr
}
