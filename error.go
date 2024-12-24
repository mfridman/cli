package cli

// NewError creates a new error with the given error code and error.
func NewError(code ErrorCode, err error) error {
	return &Error{code: code, err: err}
}

// ErrorCode represents an error code for a specific error type.
type ErrorCode int

const (
	ErrShowHelp ErrorCode = iota + 1
)

func (c ErrorCode) String() string {
	return convertErrorCode(c)
}

func convertErrorCode(code ErrorCode) string {
	switch code {
	case ErrShowHelp:
		return "show help"
	default:
		return "unknown error"
	}
}

// Error represents an error with an error code and an underlying error.
type Error struct {
	code ErrorCode
	err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.err == nil {
		return convertErrorCode(e.code) + ": <nil>"
	}
	return e.err.Error()
}
