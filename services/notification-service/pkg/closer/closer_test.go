package closer

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestStackCloseAllClosesInReverseOrder(t *testing.T) {
	stack := New()
	var closed []string

	stack.Add("first", func(context.Context) error {
		closed = append(closed, "first")
		return nil
	})
	stack.Add("second", func(context.Context) error {
		closed = append(closed, "second")
		return nil
	})

	if err := stack.CloseAll(context.Background()); err != nil {
		t.Fatalf("CloseAll returned error: %v", err)
	}

	want := []string{"second", "first"}
	if !reflect.DeepEqual(closed, want) {
		t.Fatalf("expected close order %v, got %v", want, closed)
	}
}

func TestStackCloseAllReturnsFirstErrorAndClearsStack(t *testing.T) {
	stack := New()
	firstErr := errors.New("first close failed")
	secondErr := errors.New("second close failed")

	stack.Add("first", func(context.Context) error { return firstErr })
	stack.Add("second", func(context.Context) error { return secondErr })

	err := stack.CloseAll(context.Background())
	if err == nil || !errors.Is(err, secondErr) {
		t.Fatalf("expected first LIFO error %v, got %v", secondErr, err)
	}

	if err := stack.CloseAll(context.Background()); err != nil {
		t.Fatalf("expected cleared stack, got %v", err)
	}
}
