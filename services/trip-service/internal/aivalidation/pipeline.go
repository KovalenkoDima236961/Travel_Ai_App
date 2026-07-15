package aivalidation

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

type Pipeline struct {
	validator       AIOutputValidator
	repairClient    RepairClient
	policyEvaluator PolicyEvaluator
	cfg             Config
	log             *zap.Logger
}

func NewPipeline(
	validator AIOutputValidator,
	repairClient RepairClient,
	policyEvaluator PolicyEvaluator,
	cfg Config,
	log *zap.Logger,
) *Pipeline {
	if validator == nil {
		validator = NewValidator(cfg)
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Pipeline{
		validator:       validator,
		repairClient:    repairClient,
		policyEvaluator: policyEvaluator,
		cfg:             NormalizeConfig(cfg),
		log:             log,
	}
}

func (p *Pipeline) ValidateAndRepair(ctx context.Context, input PipelineInput) (PipelineResult, error) {
	started := time.Now()
	if !p.cfg.Enabled {
		quality := GenerationQualityMetadata{
			Status:            StatusNotValidated,
			ValidatedAt:       time.Now().UTC(),
			ValidatorVersion:  ValidatorVersion,
			MaxRepairAttempts: maxRepairAttempts(input.MaxRepairAttempts, p.cfg),
			RemainingIssues:   []ValidationIssue{},
			RepairedIssues:    []ValidationIssue{},
			Warnings:          []string{},
		}
		return PipelineResult{
			FinalOutput:       input.AIOutput,
			InitialValidation: ValidationResult{Valid: true, SaveAllowed: true, QualityStatus: StatusNotValidated},
			FinalValidation:   ValidationResult{Valid: true, SaveAllowed: true, QualityStatus: StatusNotValidated},
			GenerationQuality: quality,
			SaveAllowed:       true,
			UserMessage:       "Generated successfully.",
		}, nil
	}

	output := input.AIOutput
	validation, policyEvaluation, budgetSummary, err := p.validateOutput(ctx, input, output)
	if err != nil {
		if p.cfg.FailOpen {
			quality := p.quality(input, validation, validation, nil, nil, StatusAIOutputInvalid)
			quality.Warnings = append(quality.Warnings, "Generation validation could not be completed.")
			return PipelineResult{
				FinalOutput:       output,
				InitialValidation: validation,
				FinalValidation:   validation,
				GenerationQuality: quality,
				SaveAllowed:       true,
				UserMessage:       "Generated successfully with validation warnings.",
			}, nil
		}
		return PipelineResult{}, err
	}
	initialValidation := validation
	repairs := make([]GenerationRepairAttempt, 0)

	if validation.SaveAllowed {
		status := validation.QualityStatus
		if len(validation.Warnings) > 0 {
			recordSavedWithWarnings(input.GenerationType)
		}
		quality := p.quality(input, initialValidation, validation, repairs, repairedIssues(initialValidation.Issues, validation.Issues), status)
		result := PipelineResult{
			FinalOutput:       output,
			InitialValidation: initialValidation,
			FinalValidation:   validation,
			RepairAttempts:    0,
			Repairs:           repairs,
			GenerationQuality: quality,
			SaveAllowed:       true,
			UserMessage:       successMessage(quality),
		}
		p.logResult(input, result, time.Since(started))
		return result, nil
	}

	if !p.cfg.RepairEnabled || !input.RepairAllowed || p.repairClient == nil || len(validation.RepairableIssues) == 0 {
		status := terminalStatus(validation, false)
		quality := p.quality(input, initialValidation, validation, repairs, nil, status)
		recordBlocked(input.GenerationType, status)
		result := PipelineResult{
			FinalOutput:       output,
			InitialValidation: initialValidation,
			FinalValidation:   validation,
			GenerationQuality: quality,
			SaveAllowed:       false,
			UserMessage:       userMessageForFailure(validation),
		}
		p.logResult(input, result, time.Since(started))
		if p.cfg.FailOpen {
			result.SaveAllowed = true
			result.UserMessage = "Generated successfully with validation warnings."
			return result, nil
		}
		return result, nil
	}

	maxAttempts := maxRepairAttempts(input.MaxRepairAttempts, p.cfg)
	previousBlockingIDs := issueIDs(validation.BlockingIssues)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		recordRepairAttempt(input.GenerationType, "attempted")
		repairStarted := time.Now()
		scope := selectRepairScope(validation.RepairableIssues)
		targetIssues := targetIssuesForScope(validation.RepairableIssues, scope)
		request := RepairGenerationOutputRequest{
			GenerationType:   input.GenerationType,
			CurrentOutput:    output,
			ValidationIssues: targetIssues,
			PlanningContext: PlanningContext{
				Trip:                input.Trip,
				Route:               input.Trip.Route,
				Accommodation:       input.Trip.Accommodation,
				PlanningConstraints: input.PlanningConstraints,
				WeatherForecast:     input.WeatherForecast,
				BudgetSummary:       budgetSummary,
				WorkspacePolicy:     policyEvaluation,
			},
			RepairScope: scope,
			Constraints: RepairConstraints{
				PreserveUnaffectedDays:  scope.Type != RepairScopeFullOutput,
				PreserveUserEditedItems: true,
				OutputLanguage:          outputLanguage(input.OutputLanguage),
			},
		}
		response, err := p.repairClient.RepairGenerationOutput(ctx, request)
		duration := time.Since(repairStarted)
		if err != nil {
			recordRepairAttempt(input.GenerationType, "error")
			p.log.Warn("ai generation repair failed",
				zap.String("trip_id", input.Trip.ID.String()),
				zap.String("generation_type", string(input.GenerationType)),
				zap.Int("attempt", attempt),
				zap.Error(err))
			break
		}
		newOutput := response.RepairedOutput
		newValidation, newPolicyEvaluation, newBudgetSummary, err := p.validateOutput(ctx, input, newOutput)
		if err != nil {
			p.log.Warn("ai generation repaired output validation failed",
				zap.String("trip_id", input.Trip.ID.String()),
				zap.String("generation_type", string(input.GenerationType)),
				zap.Int("attempt", attempt),
				zap.Error(err))
			break
		}
		fixedIDs := fixedIssueIDs(validation.Issues, newValidation.Issues)
		remainingIDs := issueIDs(newValidation.Issues)
		repairs = append(repairs, GenerationRepairAttempt{
			Attempt:         attempt,
			RepairScope:     scope,
			TargetIssueIDs:  issueIDs(targetIssues),
			IssuesFixed:     fixedIDs,
			IssuesRemaining: remainingIDs,
			DurationMS:      duration.Milliseconds(),
			AIProviderMode:  p.repairClient.ProviderMode(),
			ChangesMade:     response.ChangesMade,
			Warnings:        response.Warnings,
		})
		recordRepairAttempt(input.GenerationType, "completed")

		output = newOutput
		validation = newValidation
		policyEvaluation = newPolicyEvaluation
		budgetSummary = newBudgetSummary
		if validation.SaveAllowed {
			recordRepairSuccess(input.GenerationType)
			break
		}
		currentBlockingIDs := issueIDs(validation.BlockingIssues)
		if reflect.DeepEqual(previousBlockingIDs, currentBlockingIDs) {
			p.log.Info("ai generation repair stopped without improvement",
				zap.String("trip_id", input.Trip.ID.String()),
				zap.String("generation_type", string(input.GenerationType)),
				zap.Int("attempt", attempt),
				zap.Strings("blocking_issue_ids", currentBlockingIDs))
			break
		}
		previousBlockingIDs = currentBlockingIDs
	}

	status := terminalStatus(validation, len(repairs) > 0)
	if validation.SaveAllowed {
		if len(validation.Warnings) > 0 {
			status = StatusRepairedWithWarnings
			recordSavedWithWarnings(input.GenerationType)
		} else {
			status = StatusRepairedAndValidated
		}
	} else {
		recordRepairFailure(input.GenerationType)
		recordBlocked(input.GenerationType, status)
	}
	quality := p.quality(input, initialValidation, validation, repairs, repairedIssues(initialValidation.Issues, validation.Issues), status)
	result := PipelineResult{
		FinalOutput:       output,
		InitialValidation: initialValidation,
		FinalValidation:   validation,
		RepairAttempts:    len(repairs),
		Repairs:           repairs,
		GenerationQuality: quality,
		SaveAllowed:       validation.SaveAllowed,
		UserMessage:       successOrFailureMessage(validation, quality),
	}
	p.logResult(input, result, time.Since(started))
	if !result.SaveAllowed && p.cfg.FailOpen {
		result.SaveAllowed = true
		result.UserMessage = "Generated successfully with validation warnings."
	}
	return result, nil
}

func (p *Pipeline) validateOutput(ctx context.Context, input PipelineInput, output aggregate.Itinerary) (ValidationResult, *workspacepolicies.Evaluation, *budget.Summary, error) {
	summary := budget.CalculateBudgetSummary(budget.TripBudget{
		Amount:        input.Trip.BudgetAmount,
		Currency:      input.Trip.BudgetCurrency,
		Days:          int(input.Trip.Days),
		Accommodation: input.Trip.Accommodation,
		Route:         input.Trip.Route,
	}, output)
	var policyEvaluation *workspacepolicies.Evaluation
	if p.policyEvaluator != nil {
		evaluation, err := p.policyEvaluator.EvaluatePolicyForItinerary(ctx, input.Trip, output)
		if err != nil {
			if !p.cfg.FailOpen {
				return ValidationResult{}, nil, nil, err
			}
			p.log.Warn("ai generation policy evaluation failed",
				zap.String("trip_id", input.Trip.ID.String()),
				zap.Error(err))
		} else {
			policyEvaluation = evaluation
		}
	}
	validation, err := p.validator.Validate(ctx, ValidationInput{
		GenerationType:      input.GenerationType,
		Trip:                input.Trip,
		Itinerary:           output,
		PlanningConstraints: input.PlanningConstraints,
		BudgetSummary:       &summary,
		PolicyEvaluation:    policyEvaluation,
		WeatherForecast:     input.WeatherForecast,
		Context: ValidationContext{
			ExpectedDayCount: int(input.Trip.Days),
			RepairAllowed:    input.RepairAllowed,
		},
	})
	return validation, policyEvaluation, &summary, err
}

func (p *Pipeline) quality(
	input PipelineInput,
	initial ValidationResult,
	final ValidationResult,
	repairs []GenerationRepairAttempt,
	repaired []ValidationIssue,
	status GenerationQualityStatus,
) GenerationQualityMetadata {
	counts := countIssues(final.Issues)
	return GenerationQualityMetadata{
		Status:             status,
		ValidatedAt:        time.Now().UTC(),
		ValidatorVersion:   ValidatorVersion,
		RepairAttempts:     len(repairs),
		MaxRepairAttempts:  maxRepairAttempts(input.MaxRepairAttempts, p.cfg),
		BlockingIssueCount: counts.blocking,
		CriticalIssueCount: counts.critical,
		HighIssueCount:     counts.high,
		WarningIssueCount:  counts.warning,
		RemainingIssues:    final.Issues,
		RepairedIssues:     repaired,
		Warnings:           final.Warnings,
		RepairAttemptLog:   repairs,
	}
}

func (p *Pipeline) logResult(input PipelineInput, result PipelineResult, duration time.Duration) {
	fields := []zap.Field{
		zap.String("trip_id", input.Trip.ID.String()),
		zap.String("generation_type", string(input.GenerationType)),
		zap.String("quality_status", string(result.GenerationQuality.Status)),
		zap.Int("repair_attempts", result.RepairAttempts),
		zap.Bool("save_allowed", result.SaveAllowed),
		zap.Int("issue_count", len(result.FinalValidation.Issues)),
		zap.Int("blocking_issue_count", len(result.FinalValidation.BlockingIssues)),
		zap.Int64("duration_ms", duration.Milliseconds()),
	}
	if input.JobID != nil {
		fields = append(fields, zap.String("job_id", input.JobID.String()))
	}
	p.log.Info("ai generation validation completed", fields...)
}

type issueCounts struct {
	blocking int
	critical int
	high     int
	warning  int
}

func countIssues(issues []ValidationIssue) issueCounts {
	var out issueCounts
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityBlocking:
			out.blocking++
		case SeverityCritical:
			out.critical++
		case SeverityHigh:
			out.high++
		case SeverityWarning:
			out.warning++
		}
	}
	return out
}

func terminalStatus(validation ValidationResult, attemptedRepair bool) GenerationQualityStatus {
	if validation.SaveAllowed {
		if attemptedRepair {
			if len(validation.Warnings) > 0 {
				return StatusRepairedWithWarnings
			}
			return StatusRepairedAndValidated
		}
		if len(validation.Warnings) > 0 {
			return StatusValidatedWithWarnings
		}
		return StatusValidated
	}
	if attemptedRepair {
		return StatusRepairFailed
	}
	for _, issue := range validation.BlockingIssues {
		if issue.Category == CategorySchema {
			return StatusSchemaInvalid
		}
		if issue.Category == CategoryPolicy && issue.Severity == SeverityBlocking {
			return StatusBlockedByPolicy
		}
	}
	return StatusBlockedByCriticalIssues
}

func successMessage(quality GenerationQualityMetadata) string {
	switch quality.Status {
	case StatusValidatedWithWarnings:
		return pluralWarnings("Generated successfully with", quality.WarningIssueCount)
	case StatusRepairedAndValidated:
		return "Generated successfully. AI repaired scheduling issues."
	case StatusRepairedWithWarnings:
		return pluralWarnings("Generated successfully with", quality.WarningIssueCount)
	default:
		return "Generated successfully."
	}
}

func successOrFailureMessage(validation ValidationResult, quality GenerationQualityMetadata) string {
	if validation.SaveAllowed {
		return successMessage(quality)
	}
	if quality.Status == StatusRepairFailed {
		return "AI repair could not fix the schedule conflict. Try relaxing constraints or regenerating the affected day."
	}
	return userMessageForFailure(validation)
}

func pluralWarnings(prefix string, count int) string {
	if count == 1 {
		return prefix + " 1 warning."
	}
	return prefix + " " + strconv.Itoa(count) + " warnings."
}

func maxRepairAttempts(value int, cfg Config) int {
	if value > 0 {
		return value
	}
	if cfg.MaxRepairAttempts > 0 {
		return cfg.MaxRepairAttempts
	}
	return 2
}

func outputLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "es", "uk", "fr":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "en"
	}
}

func repairedIssues(initial, final []ValidationIssue) []ValidationIssue {
	finalSet := issueIDSet(final)
	out := make([]ValidationIssue, 0)
	for _, issue := range initial {
		if _, ok := finalSet[issue.ID]; !ok {
			out = append(out, issue)
		}
	}
	return out
}

func fixedIssueIDs(previous, current []ValidationIssue) []string {
	out := issueIDs(repairedIssues(previous, current))
	return out
}

func selectRepairScope(issues []ValidationIssue) RepairScope {
	if len(issues) == 0 {
		return RepairScope{Type: RepairScopeFullOutput}
	}
	var (
		daySet      = map[int]struct{}{}
		itemSet     = map[[2]int]struct{}{}
		routeLegSet = map[string]struct{}{}
		policyOnly  = true
		budgetOnly  = true
	)
	for _, issue := range issues {
		if issue.DayNumber != nil {
			daySet[*issue.DayNumber] = struct{}{}
			if issue.ItemIndex != nil {
				itemSet[[2]int{*issue.DayNumber, *issue.ItemIndex}] = struct{}{}
			}
		}
		if issue.RouteLegID != "" {
			routeLegSet[issue.RouteLegID] = struct{}{}
		}
		if issue.Category != CategoryPolicy {
			policyOnly = false
		}
		if issue.Category != CategoryBudget {
			budgetOnly = false
		}
		if issue.Category == CategorySchema && issue.DayNumber == nil {
			return RepairScope{Type: RepairScopeFullOutput}
		}
	}
	if policyOnly {
		return RepairScope{Type: RepairScopePolicyIssues}
	}
	if budgetOnly {
		return RepairScope{Type: RepairScopeBudgetSection}
	}
	if len(routeLegSet) == 1 && len(daySet) > 1 {
		for legID := range routeLegSet {
			return RepairScope{Type: RepairScopeRouteLeg, RouteLegID: legID}
		}
	}
	if len(daySet) == 1 {
		for day := range daySet {
			dayNumber := day
			if len(itemSet) == 1 {
				for encoded := range itemSet {
					itemIndex := encoded[1]
					return RepairScope{Type: RepairScopeItem, DayNumber: &dayNumber, ItemIndex: &itemIndex}
				}
			}
			return RepairScope{Type: RepairScopeDay, DayNumber: &dayNumber}
		}
	}
	return RepairScope{Type: RepairScopeSelectedIssues}
}

func targetIssuesForScope(issues []ValidationIssue, scope RepairScope) []ValidationIssue {
	if scope.Type == RepairScopeFullOutput || scope.Type == RepairScopeSelectedIssues || scope.Type == RepairScopePolicyIssues || scope.Type == RepairScopeBudgetSection {
		return issues
	}
	out := make([]ValidationIssue, 0)
	for _, issue := range issues {
		if scope.RouteLegID != "" && issue.RouteLegID == scope.RouteLegID {
			out = append(out, issue)
			continue
		}
		if scope.DayNumber != nil && issue.DayNumber != nil && *scope.DayNumber == *issue.DayNumber {
			if scope.ItemIndex == nil || issue.ItemIndex == nil || *scope.ItemIndex == *issue.ItemIndex {
				out = append(out, issue)
			}
		}
	}
	if len(out) == 0 {
		return issues
	}
	return out
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}
