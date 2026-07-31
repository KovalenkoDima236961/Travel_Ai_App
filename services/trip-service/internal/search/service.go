package search

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

type Repository interface {
	Search(context.Context, RepositorySearchParams) ([]Result, error)
}

type WorkspaceProvider interface {
	ListForUser(context.Context, uuid.UUID) ([]workspaces.UserWorkspace, error)
	BatchInfo(context.Context, []uuid.UUID) ([]workspaces.WorkspaceInfo, error)
}

type Service struct {
	repo       Repository
	workspaces WorkspaceProvider
	cfg        Config
	log        *zap.Logger
}

func NewService(repo Repository, workspaceProvider WorkspaceProvider, cfg Config, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		repo:       repo,
		workspaces: workspaceProvider,
		cfg:        NormalizeConfig(cfg),
		log:        log,
	}
}

func (s *Service) Search(ctx context.Context, userID uuid.UUID, params Params) (Response, error) {
	started := time.Now()
	status := "ok"
	defer func() {
		if status != "ok" {
			recordSearchError(params.Scope, status)
		}
	}()

	if !s.cfg.Enabled {
		return Response{}, nil
	}
	query, err := normalizeQuery(params.Query, s.cfg.MaxQueryLength)
	if err != nil {
		status = "invalid_query"
		return Response{}, err
	}
	typeFilters, err := resultTypeFilters(params.Types)
	if err != nil {
		status = "invalid_filter"
		return Response{}, err
	}
	limit := params.Limit
	if limit <= 0 {
		limit = s.cfg.DefaultLimit
	}
	if limit > s.cfg.MaxLimit {
		limit = s.cfg.MaxLimit
	}

	tokens := tokenize(query)
	results := []Result{}
	if params.IncludeCommands {
		results = commandResults(query, tokens, params.TripID, typeFilters)
	}
	if query == "" || runeLen(query) < s.cfg.MinQueryLength {
		now := time.Now()
		for i := range results {
			results[i].MatchedFields = matchedFields(query, tokens, results[i])
			results[i].Score = scoreResult(query, tokens, results[i], params.TripID, now)
		}
		response := buildResponse(query, results, limit, s.cfg.PerCategoryLimit)
		response.QueryMeta = queryMetadata(query, params.Scope, typeFilters, params.IncludeArchived)
		recordSearch(params.Scope, "ok", time.Since(started), len(response.Items))
		return response, nil
	}

	patterns := make([]string, 0, len(tokens)+1)
	patterns = append(patterns, "%"+escapeLike(query)+"%")
	for _, token := range tokens {
		patterns = append(patterns, "%"+escapeLike(token)+"%")
	}

	workspaceIDs, workspaceRoles, workspaceNames := s.workspaceAccess(ctx, userID)
	if params.Scope == ScopeWorkspace && params.WorkspaceID != nil {
		if _, ok := workspaceRoles[*params.WorkspaceID]; !ok {
			workspaceIDs = nil
		} else {
			workspaceIDs = []uuid.UUID{*params.WorkspaceID}
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, s.cfg.QueryTimeout)
	defer cancel()

	filterTripID := params.TripID
	if params.Scope != ScopeCurrentTrip {
		filterTripID = nil
	}

	repoResults, err := s.repo.Search(timeoutCtx, RepositorySearchParams{
		UserID:           userID,
		Query:            query,
		Tokens:           tokens,
		Patterns:         patterns,
		Scope:            params.Scope,
		TripID:           filterTripID,
		WorkspaceID:      params.WorkspaceID,
		WorkspaceIDs:     workspaceIDs,
		WorkspaceNames:   workspaceNames,
		CurrentTripID:    params.TripID,
		TypeFilters:      typeFilters,
		Limit:            limit,
		PerCategoryLimit: s.cfg.PerCategoryLimit,
		IncludeArchived:  params.IncludeArchived,
	})
	if err != nil {
		status = "repository"
		s.log.Warn("global search failed",
			zap.String("scope", string(params.Scope)),
			zap.Int("queryLen", len(query)),
			zap.Error(err),
		)
		return Response{}, err
	}
	results = append(results, repoResults...)

	if (params.Scope == ScopeAll || params.Scope == ScopeWorkspace) && typeAllowed(typeFilters, ResultTypeWorkspace) {
		results = append(results, s.workspaceResults(query, tokens, workspaceIDs, workspaceRoles, workspaceNames, params.WorkspaceID)...)
	}

	now := time.Now()
	for i := range results {
		results[i].MatchedFields = matchedFields(query, tokens, results[i])
		results[i].Score = scoreResult(query, tokens, results[i], params.TripID, now)
	}
	response := buildResponse(query, results, limit, s.cfg.PerCategoryLimit)
	response.QueryMeta = queryMetadata(query, params.Scope, typeFilters, params.IncludeArchived)
	recordSearch(params.Scope, "ok", time.Since(started), len(response.Items))
	return response, nil
}

func (s *Service) workspaceAccess(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, map[uuid.UUID]workspaces.Role, map[uuid.UUID]string) {
	roles := map[uuid.UUID]workspaces.Role{}
	names := map[uuid.UUID]string{}
	if s.workspaces == nil {
		return nil, roles, names
	}
	memberships, err := s.workspaces.ListForUser(ctx, userID)
	if err != nil {
		s.log.Warn("workspace list unavailable for search", zap.Error(err))
		return nil, roles, names
	}
	ids := make([]uuid.UUID, 0, len(memberships))
	for _, item := range memberships {
		ids = append(ids, item.ID)
		roles[item.ID] = item.Role
	}
	if len(ids) == 0 {
		return ids, roles, names
	}
	infos, err := s.workspaces.BatchInfo(ctx, ids)
	if err != nil {
		s.log.Warn("workspace names unavailable for search", zap.Int("workspaceCount", len(ids)), zap.Error(err))
		return ids, roles, names
	}
	for _, info := range infos {
		if !info.Archived {
			names[info.ID] = info.Name
		}
	}
	return ids, roles, names
}

func (s *Service) workspaceResults(
	query string,
	tokens []string,
	workspaceIDs []uuid.UUID,
	roles map[uuid.UUID]workspaces.Role,
	names map[uuid.UUID]string,
	workspaceFilter *uuid.UUID,
) []Result {
	results := make([]Result, 0)
	for _, workspaceID := range workspaceIDs {
		if workspaceFilter != nil && *workspaceFilter != workspaceID {
			continue
		}
		name := names[workspaceID]
		if name == "" {
			continue
		}
		if !matchesTokens(query, tokens, name, string(roles[workspaceID])) {
			continue
		}
		id := workspaceID
		results = append(results, newResult(
			ResultTypeWorkspace,
			"workspace:"+workspaceID.String(),
			name,
			"Workspace · "+string(roles[workspaceID]),
			"",
			name,
			workspaceHref(workspaceID),
			idMetadata(map[string]string{
				"workspaceId": workspaceID.String(),
				"role":        string(roles[workspaceID]),
			}),
			resultRefs{WorkspaceID: &id},
		))
	}
	return results
}

func queryMetadata(
	query string,
	scope Scope,
	typeFilters map[ResultType]struct{},
	includeArchived bool,
) QueryMetadata {
	types := make([]ResultType, 0, len(typeFilters))
	for resultType := range typeFilters {
		types = append(types, resultType)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	return QueryMetadata{
		Normalized:      query,
		Scope:           scope,
		Types:           types,
		IncludeArchived: includeArchived,
	}
}
