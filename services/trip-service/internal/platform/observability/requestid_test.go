package observability

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextWithRequestIDsPreservesExplicitValues(t *testing.T) {
	ctx := ContextWithRequestIDs(context.Background(), "request-1", "correlation-1")

	if got := RequestIDFromContext(ctx); got != "request-1" {
		t.Fatalf("request id = %q, want request-1", got)
	}
	if got := CorrelationIDFromContext(ctx); got != "correlation-1" {
		t.Fatalf("correlation id = %q, want correlation-1", got)
	}
}

func TestContextWithRequestIDsGeneratesMissingValues(t *testing.T) {
	ctx := ContextWithRequestIDs(context.Background(), "", "")

	requestID := RequestIDFromContext(ctx)
	correlationID := CorrelationIDFromContext(ctx)
	if requestID == "" {
		t.Fatal("expected generated request id")
	}
	if correlationID != requestID {
		t.Fatalf("correlation id = %q, want request id %q", correlationID, requestID)
	}
}

func TestRequestIDMiddlewareGeneratesIDsAndSetsResponseHeaders(t *testing.T) {
	var seenRequestID string
	var seenCorrelationID string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRequestID = RequestIDFromContext(r.Context())
		seenCorrelationID = CorrelationIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if seenRequestID == "" {
		t.Fatal("handler did not receive request id")
	}
	if seenCorrelationID != seenRequestID {
		t.Fatalf("correlation id = %q, want request id %q", seenCorrelationID, seenRequestID)
	}
	if got := rr.Header().Get(HeaderRequestID); got != seenRequestID {
		t.Fatalf("response request id = %q, want %q", got, seenRequestID)
	}
	if got := rr.Header().Get(HeaderCorrelationID); got != seenCorrelationID {
		t.Fatalf("response correlation id = %q, want %q", got, seenCorrelationID)
	}
}

func TestRequestIDMiddlewarePreservesInboundIDs(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != "inbound-request" {
			t.Fatalf("request id = %q, want inbound-request", got)
		}
		if got := CorrelationIDFromContext(r.Context()); got != "inbound-correlation" {
			t.Fatalf("correlation id = %q, want inbound-correlation", got)
		}
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/trips", nil)
	req.Header.Set(HeaderRequestID, "inbound-request")
	req.Header.Set(HeaderCorrelationID, "inbound-correlation")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get(HeaderRequestID); got != "inbound-request" {
		t.Fatalf("response request id = %q, want inbound-request", got)
	}
	if got := rr.Header().Get(HeaderCorrelationID); got != "inbound-correlation" {
		t.Fatalf("response correlation id = %q, want inbound-correlation", got)
	}
}

func TestRequestIDRoundTripperPropagatesHeaders(t *testing.T) {
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get(HeaderRequestID); got != "request-1" {
			t.Fatalf("request id header = %q, want request-1", got)
		}
		if got := req.Header.Get(HeaderCorrelationID); got != "correlation-1" {
			t.Fatalf("correlation id header = %q, want correlation-1", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(http.NoBody),
			Header:     make(http.Header),
		}, nil
	})

	client := &http.Client{Transport: NewRequestIDRoundTripper(base)}
	ctx := ContextWithRequestIDs(context.Background(), "request-1", "correlation-1")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/resource", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	_ = resp.Body.Close()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
