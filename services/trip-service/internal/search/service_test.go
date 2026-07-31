package search

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type captureRepository struct {
	calls  int
	params RepositorySearchParams
}

func (r *captureRepository) Search(_ context.Context, params RepositorySearchParams) ([]Result, error) {
	r.calls++
	r.params = params
	return nil, nil
}

func TestNormalizeQueryCollapsesWhitespaceAndRejectsUnsafeInput(t *testing.T) {
	query, err := normalizeQuery("  Rome   Colosseum  ", 200)
	if err != nil {
		t.Fatalf("normalize query: %v", err)
	}
	if query != "Rome Colosseum" {
		t.Fatalf("unexpected normalized query %q", query)
	}

	if _, err := normalizeQuery("Rome\nColosseum", 200); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected invalid query for control character, got %v", err)
	}
	if _, err := normalizeQuery("abcdef", 5); !errors.Is(err, ErrQueryTooLong) {
		t.Fatalf("expected query too long, got %v", err)
	}
}

func TestSearchEmptyQueryReturnsCommandsWithoutRepositoryCall(t *testing.T) {
	repo := &captureRepository{}
	svc := NewService(repo, nil, Config{Enabled: true, DefaultLimit: 20}, nil)

	response, err := svc.Search(context.Background(), uuid.New(), Params{
		Query:           "  ",
		Scope:           ScopeAll,
		IncludeCommands: true,
	})
	if err != nil {
		t.Fatalf("search empty query: %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("expected no repository call for empty query, got %d", repo.calls)
	}
	if len(response.Items) == 0 {
		t.Fatalf("expected safe command results for empty query")
	}
	if response.Query != "" || response.QueryMeta.Normalized != "" {
		t.Fatalf("unexpected query metadata: %#v", response.QueryMeta)
	}
}

func TestSearchPropagatesTypeAndArchiveFilters(t *testing.T) {
	repo := &captureRepository{}
	svc := NewService(repo, nil, Config{Enabled: true, DefaultLimit: 20}, nil)
	userID := uuid.New()

	_, err := svc.Search(context.Background(), userID, Params{
		Query:           "Rome",
		Scope:           ScopeAll,
		Types:           []ResultType{ResultTypeTrip},
		IncludeArchived: true,
	})
	if err != nil {
		t.Fatalf("search with filters: %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("expected one repository call, got %d", repo.calls)
	}
	if !repo.params.IncludeArchived {
		t.Fatalf("expected include archived to propagate")
	}
	if !typeAllowed(repo.params.TypeFilters, ResultTypeTrip) {
		t.Fatalf("expected trip type filter to propagate")
	}
	if typeAllowed(repo.params.TypeFilters, ResultTypeExpense) {
		t.Fatalf("did not expect expense type to be allowed")
	}
}

func TestSearchRejectsUnknownTypeFilter(t *testing.T) {
	svc := NewService(&captureRepository{}, nil, Config{Enabled: true}, nil)

	_, err := svc.Search(context.Background(), uuid.New(), Params{
		Query: "Rome",
		Scope: ScopeAll,
		Types: []ResultType{"secret"},
	})
	if !errors.Is(err, ErrInvalidFilter) {
		t.Fatalf("expected invalid filter error, got %v", err)
	}
}
