package cmd

// exitCodeError lets a command handler attach a specific process exit code
// to the error it returns from RunE. Execute() unwraps it via errors.As.
//
// Use the helpers below rather than constructing the struct directly.
type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string { return e.err.Error() }

func (e *exitCodeError) Unwrap() error { return e.err }

// usageError wraps err so Execute() exits with the usage/config exit code (2).
func usageError(err error) error {
	return &exitCodeError{code: exitUsageError, err: err}
}
