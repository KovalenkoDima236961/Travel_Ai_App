package rabbitmq

import "testing"

func TestNewManagementClientRejectsInvalidURL(t *testing.T) {
	if _, err := NewManagementClient(ManagementConfig{URL: "amqp://rabbitmq:5672"}); err == nil {
		t.Fatal("expected non-http management URL to fail")
	}
}

func TestVHostFromAMQPURL(t *testing.T) {
	cases := map[string]string{
		"amqp://guest:guest@rabbitmq:5672/":        "/",
		"amqp://guest:guest@rabbitmq:5672/travel":  "travel",
		"amqp://guest:guest@rabbitmq:5672/%2Fprod": "/prod",
		"not-a-url": "/",
	}
	for raw, want := range cases {
		if got := vhostFromAMQPURL(raw); got != want {
			t.Fatalf("vhostFromAMQPURL(%q) = %q, want %q", raw, got, want)
		}
	}
}
