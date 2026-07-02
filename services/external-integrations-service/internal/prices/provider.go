package prices

import "context"

type PriceProvider interface {
	EstimatePrice(ctx context.Context, input PriceEstimateInput) (*PriceEstimateResult, error)
}
