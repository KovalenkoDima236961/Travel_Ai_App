package logger

import "testing"

func TestInitLoggerReturnsCachedLogger(t *testing.T) {
	first := InitLogger()
	second := InitLogger()

	if first == nil {
		t.Fatal("expected logger")
	}
	if second != first {
		t.Fatal("expected InitLogger to return the cached logger")
	}
}
