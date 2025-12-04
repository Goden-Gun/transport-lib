package codes

// ErrorCode represents structured transport errors shared across services.
type ErrorCode struct {
	Numeric int32
	Symbol  string
	Message string
}

var (
	// ErrUnauthorized indicates token verification failure.
	ErrUnauthorized = ErrorCode{Numeric: 40101, Symbol: "TOKEN_INVALID", Message: "authentication failed"}
	// ErrPermissionDenied indicates user lacks capability.
	ErrPermissionDenied = ErrorCode{Numeric: 40301, Symbol: "PERMISSION_DENIED", Message: "permission denied"}
	// ErrInvalidPayload indicates malformed request payload.
	ErrInvalidPayload = ErrorCode{Numeric: 41001, Symbol: "INVALID_PAYLOAD", Message: "invalid payload"}
	// ErrTooManyRequests indicates rate limiting.
	ErrTooManyRequests = ErrorCode{Numeric: 42901, Symbol: "RATE_LIMITED", Message: "too many requests"}
	// ErrInternal indicates unknown server error.
	ErrInternal = ErrorCode{Numeric: 50001, Symbol: "INTERNAL_ERROR", Message: "internal server error"}
)

// Registry exposes a static list for validation or docs.
var Registry = []ErrorCode{
	ErrUnauthorized,
	ErrPermissionDenied,
	ErrInvalidPayload,
	ErrTooManyRequests,
	ErrInternal,
}
