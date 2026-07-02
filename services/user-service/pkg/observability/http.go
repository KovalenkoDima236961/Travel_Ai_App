package observability

import "net/http"

type requestIDRoundTripper struct {
	base http.RoundTripper
}

func NewRequestIDRoundTripper(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return requestIDRoundTripper{base: base}
}

func (t requestIDRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	PropagateRequestIDs(clone)
	return t.base.RoundTrip(clone)
}

func InstrumentHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	copy := *client
	copy.Transport = NewRequestIDRoundTripper(base)
	return &copy
}
