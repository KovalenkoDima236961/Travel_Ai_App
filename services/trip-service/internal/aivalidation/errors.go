package aivalidation

import "fmt"

const (
	ErrorCodeSchemaInvalid     = "ai_generation_schema_invalid"
	ErrorCodeValidationFailed  = "ai_generation_validation_failed"
	ErrorCodeRepairFailed      = "ai_generation_repair_failed"
	ErrorCodeBlockedByPolicy   = "ai_generation_blocked_by_policy"
	ErrorCodeRouteConflict     = "ai_generation_route_conflict"
	ErrorCodeTransportConflict = "ai_generation_transport_conflict"
	ErrorCodeBudgetConflict    = "ai_generation_budget_conflict"
	ErrorCodeOutputInvalid     = "ai_output_invalid"
)

type ValidationError struct {
	Code    string
	Message string
	Issues  []ValidationIssue
	Quality GenerationQualityMetadata
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func NewValidationError(result PipelineResult) *ValidationError {
	code := errorCodeForQuality(result.GenerationQuality)
	message := result.UserMessage
	if message == "" {
		message = userMessageForFailure(result.FinalValidation)
	}
	return &ValidationError{
		Code:    code,
		Message: message,
		Issues:  result.FinalValidation.BlockingIssues,
		Quality: result.GenerationQuality,
	}
}

func errorCodeForQuality(quality GenerationQualityMetadata) string {
	for _, issue := range quality.RemainingIssues {
		switch issue.Category {
		case CategorySchema:
			return ErrorCodeSchemaInvalid
		case CategoryPolicy:
			if issue.Severity == SeverityBlocking {
				return ErrorCodeBlockedByPolicy
			}
		case CategoryRoute:
			return ErrorCodeRouteConflict
		case CategoryTransport:
			return ErrorCodeTransportConflict
		case CategoryBudget:
			if issue.Severity == SeverityBlocking || issue.Severity == SeverityCritical {
				return ErrorCodeBudgetConflict
			}
		}
	}
	switch quality.Status {
	case StatusSchemaInvalid:
		return ErrorCodeSchemaInvalid
	case StatusBlockedByPolicy:
		return ErrorCodeBlockedByPolicy
	case StatusRepairFailed:
		return ErrorCodeRepairFailed
	case StatusAIOutputInvalid:
		return ErrorCodeOutputInvalid
	default:
		return ErrorCodeValidationFailed
	}
}

func userMessageForFailure(validation ValidationResult) string {
	for _, issue := range validation.BlockingIssues {
		switch issue.Category {
		case CategoryPolicy:
			return "The generated plan violated a blocking workspace policy."
		case CategoryRoute:
			return "The itinerary could not be saved because it conflicts with your selected route."
		case CategoryTransport:
			return "The generated itinerary conflicts with your selected transport times."
		case CategorySchema:
			return "The AI generated an itinerary with an invalid structure."
		}
	}
	if len(validation.BlockingIssues) > 0 {
		return fmt.Sprintf("The itinerary could not be saved because %d critical issue(s) remain.", len(validation.BlockingIssues))
	}
	return "The itinerary could not be saved after validation."
}
