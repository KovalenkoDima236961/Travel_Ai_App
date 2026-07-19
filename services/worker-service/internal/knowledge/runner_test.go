package knowledge

import "testing"

func TestRunnerRejectsMissingStore(t *testing.T) {
	var runner *Runner
	if _, err := runner.Run(t.Context(), Request{}); err == nil {
		t.Fatal("expected runner without store to fail")
	}
}
