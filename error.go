package application

import "strings"

// Error for modules containing multiple errors
type Error struct {
	Errors []error
}

// Append the error to the existing one, when the target is nil, it will create a new Error struct
func (e *Error) Append(err error) *Error {
	if err == nil {
		return e
	}

	if e == nil {
		e = new(Error)
	}

	switch errCast := err.(type) {
	case *Error:
		for _, subErr := range errCast.Errors {
			switch subErr := subErr.(type) {
			case *Error:
				if subErr != nil {
					e.Errors = append(e.Errors, subErr.Errors...)
				}
			default:
				if subErr != nil {
					e.Errors = append(e.Errors, subErr)
				}
			}
		}
	default:
		e.Errors = append(e.Errors, errCast)
	}

	return e
}

func (e Error) Error() string {
	// not the best formatter, but it works
	if len(e.Errors) == 0 {
		return ""
	}

	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	result := "multiple errors: "
	for _, err := range e.Errors {
		result += err.Error() + "; "
	}

	return strings.TrimSpace(result)
}

// Err returns an error if any exist, or nil
func (e Error) Err() error {
	if len(e.Errors) == 0 {
		return nil
	}

	return e
}

// Unwrap the error
func (e Error) Unwrap() error {
	if len(e.Errors) == 0 {
		return e
	}

	// for now, just return index 0
	return e.Errors[0]
}
