package observability

import "testing"

func TestProviderCircuitOpensAfterRepeatedFailuresAndResetsOnSuccess(t *testing.T) {
	provider, operation := "test-provider", "test-operation"
	providerCircuits.Delete(provider + ":" + operation)

	for i := 0; i < providerCircuitFailureThreshold; i++ {
		if !ProviderCircuitAllows(provider, operation) {
			t.Fatalf("circuit opened before failure %d", i+1)
		}
		RecordProviderCircuitFailure(provider, operation)
	}
	if ProviderCircuitAllows(provider, operation) {
		t.Fatal("circuit should short-circuit after the configured threshold")
	}
	RecordProviderCircuitSuccess(provider, operation)
	if !ProviderCircuitAllows(provider, operation) {
		t.Fatal("a successful probe should close the circuit")
	}
}
