package postgres

import "time"

func timePtrArg(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}
