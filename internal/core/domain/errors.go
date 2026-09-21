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

	// ErrOutcomeUncertain is ADR 0004's stable code for a mutating
	// operation whose confirmed outcome cannot be reported: not a
	// rollback signal, not a successful receipt. Callers branch on this
	// Code plus the Effect/Confirmation fields below — never on Detail's
	// wording, and never on IOFailure's text (an unclassified IOFailure
	// from a mutating request must still be treated conservatively, per
	// the ADR's consequences section, until every producer is migrated).
	ErrOutcomeUncertain ErrorCode = "OutcomeUncertain"
)

// Effect describes whether the handler observed the logical change take
// effect. Never infer Applied from a timeout alone (ADR 0004).
type Effect string

const (
	// EffectApplied means the handler observed the change take effect —
	// an observation at the moment the error arose, not a guarantee it
	// survives a subsequent crash.
	EffectApplied Effect = "Applied"
	// EffectUnknown means the handler cannot establish whether the
	// change took effect at all.
	EffectUnknown Effect = "Unknown"
)

// ConfirmationKind names what is missing.
type ConfirmationKind string

const (
	// ConfirmationDurability: the change applied, but persistence
	// confirmation (e.g. a directory fsync after rename) is missing.
	ConfirmationDurability ConfirmationKind = "Durability"
	// ConfirmationOutcome: the command's response or completion itself
	// is unavailable — a lost local command response, for example. This
	// boundary cannot manufacture a server-side confirmation.
	ConfirmationOutcome ConfirmationKind = "Outcome"
)

// Error carries a stable code plus a human-readable detail. The detail is
// supplementary; callers branch on Code, never on Detail.
//
// RequestID, WorkspaceID, Effect, Confirmation, and ObservedRevision are
// populated only when Code is ErrOutcomeUncertain (ADR 0004's stable
// shape for that one code); every other code ignores them. They exist so
// a contract can assert on stable fields instead of matching Detail's
// free text substring-by-substring — exactly the fragility Quality
// flagged against the pre-ADR "commit applied but durability
// unconfirmed" text baked into IOFailure's Detail.
type Error struct {
	Code   ErrorCode
	Detail string

	RequestID        RequestID
	WorkspaceID      WorkspaceID
	Effect           Effect
	Confirmation     ConfirmationKind
	ObservedRevision *uint64
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Detail
}
