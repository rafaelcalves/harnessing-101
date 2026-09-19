package domain

// ErrorCode is a stable, transport-independent failure code. See
// boundaries.md, "Common vocabulary and compatibility".
type ErrorCode string

const (
	ErrInvalidArgument  ErrorCode = "InvalidArgument"
	ErrNotFound         ErrorCode = "NotFound"
	ErrConflict         ErrorCode = "Conflict"
	ErrDenied           ErrorCode = "Denied"
	ErrUnsupported      ErrorCode = "Unsupported"
	ErrBusy             ErrorCode = "Busy"
	ErrIOFailure        ErrorCode = "IOFailure"
	ErrCursorExpired    ErrorCode = "CursorExpired"
	ErrRecoveryRequired ErrorCode = "RecoveryRequired"
)

// Error carries a stable code plus a human-readable detail. The detail is
// supplementary; callers branch on Code, never on Detail.
type Error struct {
	Code   ErrorCode
	Detail string
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Detail
}
