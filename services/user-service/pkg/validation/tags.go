package validation

import (
	"net/url"
	"time"

	"github.com/go-playground/validator/v10"
)

// TagOption registers custom validation behavior.
type TagOption func(*Validation) error

// BeforeNowTag validates time.Time values before now.
func BeforeNowTag() TagOption {
	return func(v *Validation) error {
		return v.v.RegisterValidation("before_now", func(fl validator.FieldLevel) bool {
			if date, ok := fl.Field().Interface().(time.Time); ok {
				return date.Before(time.Now())
			}
			return false
		})
	}
}

// OriginTag validates browser origin values.
func OriginTag() TagOption {
	return func(v *Validation) error {
		return v.v.RegisterValidation("origin", originValidator)
	}
}

func originValidator(fl validator.FieldLevel) bool {
	origin := fl.Field().String()
	if origin == "*" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.Path == ""
}
