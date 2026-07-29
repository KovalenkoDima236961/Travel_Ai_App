package aimodel

import (
	"context"
	"strings"
)

type StaticRouter struct {
	cfg         Config
	environment string
	baseline    Deployment
	candidate   *Deployment
}

func NewStaticRouter(cfg Config, environment string) *StaticRouter {
	cfg = NormalizeConfig(cfg)
	env := strings.TrimSpace(environment)
	if env == "" {
		env = "local"
	}
	return &StaticRouter{
		cfg:         cfg,
		environment: env,
		baseline: Deployment{
			DeploymentKey:  cfg.DefaultDeploymentKey,
			Environment:    env,
			ModelVariant:   VariantGroundedBaseline,
			Status:         StatusActive,
			TaskType:       TaskGroundedItineraryGeneration,
			TrafficMode:    TrafficActive,
			AssignmentSalt: firstNonEmpty(cfg.DeploymentAssignmentSalt, "baseline"),
			PromptVersion:  "itinerary_generation_v1",
		},
	}
}

func (r *StaticRouter) WithCandidate(candidate Deployment) *StaticRouter {
	if r == nil {
		return nil
	}
	r.candidate = &candidate
	return r
}

func (r *StaticRouter) Decide(_ context.Context, ctx RoutingContext) (RoutingDecision, error) {
	if r == nil {
		return RoutingDecision{}, nil
	}
	if strings.TrimSpace(ctx.Environment) == "" {
		ctx.Environment = r.environment
	}
	return Decide(DecisionInput{
		Config:    r.cfg,
		Context:   ctx,
		Baseline:  r.baseline,
		Candidate: r.candidate,
	}), nil
}
