package logger

import "testing"

func TestInitLoggerReturnsSameNonNilLogger(t *testing.T) {
	first := InitLogger()
	second := InitLogger()

	if first == nil {
		t.Fatal("first logger is nil")
	}
	if second == nil {
		t.Fatal("second logger is nil")
	}
	if first != second {
		t.Fatal("InitLogger should return the singleton logger")
	}
}
