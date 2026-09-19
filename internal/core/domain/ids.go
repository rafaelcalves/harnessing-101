package domain

// Opaque, workspace-unique string identifiers. None encode a path, PID,
// display name, or timestamp; see boundaries.md port 10 (IDSource).
type (
	WorkspaceID string
	AgentID     string
	TaskID      string
	MessageID   string
	RunID       string
	OperationID string
	RequestID   string
	EventID     string
)
