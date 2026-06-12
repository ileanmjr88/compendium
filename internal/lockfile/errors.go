package lockfile

import (
	"fmt"
)

type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("invalid lockfile: %s (%s=%q)", e.Message, e.Field, e.Value)
	}
	return fmt.Sprintf("invalid lockfile: %s (%s)", e.Message, e.Field)
}
