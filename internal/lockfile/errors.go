package lockfile

import (
	"fmt"
)

type SchemaVersionError struct {
	Path  string // lockfile path, for context
	Found int    // schema declared in the file
	Max   int    // CurrentSchema
}

func (e *SchemaVersionError) Error() string {
	return fmt.Sprintf(
		"%s: lockfile schema %d in newer supported schema %d; upgrade compendium",
		e.Path, e.Found, e.Max)
}

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
