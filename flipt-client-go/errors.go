package flipt

// ErrorCode represents a category of error for the Flipt client.
type ErrorCode string

const (
	// ErrorCodeNetwork represents an error that occurred during a network request.
	ErrorCodeNetwork ErrorCode = "network"
)

// Error is the interface implemented by all Flipt client errors.
type Error interface {
	error
	Code() ErrorCode
}

// networkError represents an error that occurred during an HTTP request.
type networkError struct {
	msg    string
	status int
}

// Error returns the error message.
func (e *networkError) Error() string {
	return e.msg
}

// Code returns the error code.
func (e *networkError) Code() ErrorCode {
	return ErrorCodeNetwork
}

// Status returns the HTTP status code.
func (e *networkError) Status() int {
	return e.status
}

// Unwrap returns nil.
func (e *networkError) Unwrap() error {
	return nil
}
