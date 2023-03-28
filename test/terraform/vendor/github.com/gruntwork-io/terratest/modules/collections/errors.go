package collections

import "fmt"

// SliceValueNotFoundError is returned when a provided values file input is not found on the host path.
type SliceValueNotFoundError struct {
	sourceString string
}

func (err SliceValueNotFoundError) Error() string {
	return fmt.Sprintf("Could not resolve requested slice value from string %s", err.sourceString)
}

// NewSliceValueNotFoundError creates a new slice found error
func NewSliceValueNotFoundError(sourceString string) SliceValueNotFoundError {
	return SliceValueNotFoundError{sourceString}
}
