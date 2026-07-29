package alpha

import "fmt"

type InputError struct {
	Message string
}

func (e *InputError) Error() string {
	if e == nil || e.Message == "" {
		return ErrInvalidInput.Error()
	}
	return e.Message
}

func invalidInput(format string, args ...any) error {
	return &InputError{Message: fmt.Sprintf(format, args...)}
}
