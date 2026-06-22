package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator interface {
	Validate(s any) error
}

// ValidationError preserves per-field validation details so handlers can
// return structured field-level error responses instead of a flat string.
type ValidationError struct {
	fields map[string]string
}

func (e *ValidationError) Fields() map[string]string { return e.fields }

func (e *ValidationError) Error() string {
	msgs := make([]string, 0, len(e.fields))
	for f, m := range e.fields {
		msgs = append(msgs, f+": "+m)
	}
	return strings.Join(msgs, "; ")
}

type Validation struct {
	v *validator.Validate
}

func NewValidator(tags ...TagOption) (*Validation, error) {
	const op = "NewValidator"
	v := &Validation{v: validator.New()}

	for _, tag := range tags {
		if err := tag(v); err != nil {
			return nil, fmt.Errorf("%s: failed to apply options for Validator: %w", op, err)
		}
	}

	return v, nil
}

func (v *Validation) Validate(s any) error {
	if err := v.v.Struct(s); err != nil {
		var invalidErr *validator.InvalidValidationError
		if errors.As(err, &invalidErr) {
			return fmt.Errorf("validator misconfigured: %w", invalidErr)
		}

		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			fields := make(map[string]string, len(ve))
			for _, fe := range ve {
				fields[fe.Field()] = fieldErrorMessage(fe)
			}
			return &ValidationError{fields: fields}
		}
	}
	return nil
}

func fieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("'%s' is required", field)
	case "required_if":
		return fmt.Sprintf("'%s' is required when %s", field, fe.Param())
	case "required_without":
		return fmt.Sprintf("'%s' is required when '%s' is not provided", field, fe.Param())
	case "min":
		return fmt.Sprintf("'%s' must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("'%s' must be at most %s characters", field, fe.Param())
	case "gte":
		return fmt.Sprintf("'%s' must be >= %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("'%s' must be <= %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("'%s' must be > %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("'%s' must be < %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("'%s' must be one of: %s", field, fe.Param())
	case "uuid", "uuid4":
		return fmt.Sprintf("'%s' must be a valid UUID", field)
	case "email":
		return fmt.Sprintf("'%s' must be a valid email address", field)
	case "datetime":
		return fmt.Sprintf("'%s' must be a valid datetime in format '%s'", field, fe.Param())
	default:
		return fmt.Sprintf("'%s' failed '%s' validation", field, fe.Tag())
	}
}
