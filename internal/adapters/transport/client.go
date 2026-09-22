package transport

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// IDSource generates the request/session identifiers Client needs.
// internal/adapters/idsource.Random{} satisfies this with zero gated
// imports; Client takes the interface rather than the concrete type so
// it stays independent of that adapter's own package (same reasoning as
// duplicating ErrorInfo instead of importing throwawayadapter).
type IDSource interface {
	NewID(ctx context.Context) (string, error)
}

// Client is one attached session's whole view of the file transport. It
// implements api.FrontendSession by writing an Envelope into its own
// intake directory and polling for the matching Response — the CLI side
// of ADR 0005's bounded local file transport. It holds no domain
// authority: every method here is a passive proxy for whatever the host
// process's Dispatch already decided.
type Client struct {
	Root         string
	SessionID    string
	IDs          IDSource
	PollInterval time.Duration
}

// Attach creates this client's session directory and blocks until the
// host binds it (or ctx ends first). callerAgentID confers no authority
// by itself — see AttachPayload's doc comment.
func Attach(ctx context.Context, root string, ids IDSource, callerAgentID domain.AgentID) (*Client, error) {
	sessionID, err := ids.NewID(ctx)
	if err != nil {
		return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	c := &Client{Root: root, SessionID: sessionID, IDs: ids}
	payload, err := json.Marshal(AttachPayload{CallerAgentID: callerAgentID})
	if err != nil {
		return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if _, err := c.roundTrip(ctx, attachOperation, attachOperation, payload); err != nil {
		return nil, err
	}
	return c, nil
}

// Detach answers the client's own request-directory cleanup: it tells
// the host this session is over, waits for the ack, then removes its
// session directory. The client that created the directory is the one
// that deletes it, so the host never has to guess whether a directory
// it can still see is one a client is still polling.
func (c *Client) Detach(ctx context.Context) error {
	_, err := c.roundTrip(ctx, detachOperation, detachOperation, nil)
	removeErr := os.RemoveAll(sessionDir(c.Root, c.SessionID))
	if err != nil {
		return err
	}
	if removeErr != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: removeErr.Error()}
	}
	return nil
}

func (c *Client) call(ctx context.Context, operation string, payload interface{}, out interface{}) error {
	requestID, err := c.IDs.NewID(ctx)
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	result, err := c.roundTrip(ctx, operation, requestID, encoded)
	if err != nil {
		return err
	}
	if out == nil || result == nil {
		return nil
	}
	if err := json.Unmarshal(result, out); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return nil
}

// roundTrip writes one envelope and polls for its response. Every
// public Client method funnels through here, including attach/detach —
// they are ordinary envelopes at the wire level, just reserved operation
// names Dispatch never sees (Host answers them itself).
func (c *Client) roundTrip(ctx context.Context, operation, requestID string, payload json.RawMessage) (json.RawMessage, error) {
	if err := validID(requestID); err != nil {
		return nil, err
	}
	envelope := Envelope{Operation: operation, Payload: payload}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := writeAtomic(intakeDir(c.Root, c.SessionID), requestPath(c.Root, c.SessionID, requestID), encoded); err != nil {
		return nil, err
	}

	interval := c.PollInterval
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		data, ok, err := readBounded(responsePath(c.Root, c.SessionID, requestID))
		if err != nil {
			return nil, err
		}
		if ok {
			var resp Response
			if err := json.Unmarshal(data, &resp); err != nil {
				return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
			}
			if !resp.OK {
				return nil, resp.Error.ToDomainError()
			}
			return resp.Result, nil
		}
		select {
		case <-ctx.Done():
			return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: ctx.Err().Error()}
		case <-ticker.C:
		}
	}
}

func (c *Client) RegisterAgent(ctx context.Context, req api.RegisterAgentRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "RegisterAgent", req, &out)
	return out, err
}

func (c *Client) UpdateAgent(ctx context.Context, req api.UpdateAgentRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "UpdateAgent", req, &out)
	return out, err
}

func (c *Client) CreateTask(ctx context.Context, req api.CreateTaskRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "CreateTask", req, &out)
	return out, err
}

func (c *Client) TransitionTask(ctx context.Context, req api.TransitionTaskRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "TransitionTask", req, &out)
	return out, err
}

func (c *Client) ReportTaskResult(ctx context.Context, req api.ReportTaskResultRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "ReportTaskResult", req, &out)
	return out, err
}

func (c *Client) AcceptTaskResult(ctx context.Context, req api.AcceptTaskResultRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "AcceptTaskResult", req, &out)
	return out, err
}

func (c *Client) RejectTaskResult(ctx context.Context, req api.RejectTaskResultRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "RejectTaskResult", req, &out)
	return out, err
}

func (c *Client) SendMessage(ctx context.Context, req api.SendMessageRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "SendMessage", req, &out)
	return out, err
}

func (c *Client) AcknowledgeMessage(ctx context.Context, req api.AcknowledgeMessageRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "AcknowledgeMessage", req, &out)
	return out, err
}

func (c *Client) ApproveProfile(ctx context.Context, req api.ApproveProfileRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "ApproveProfile", req, &out)
	return out, err
}

func (c *Client) StartRun(ctx context.Context, req api.StartRunRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "StartRun", req, &out)
	return out, err
}

func (c *Client) StopRun(ctx context.Context, req api.StopRunRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "StopRun", req, &out)
	return out, err
}

func (c *Client) SetRunBudget(ctx context.Context, req api.SetRunBudgetRequest) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "SetRunBudget", req, &out)
	return out, err
}

func (c *Client) GetAgent(ctx context.Context, id domain.AgentID) (domain.Agent, error) {
	var out domain.Agent
	err := c.call(ctx, "GetAgent", struct {
		AgentID domain.AgentID `json:"agentID"`
	}{id}, &out)
	return out, err
}

func (c *Client) GetTask(ctx context.Context, id domain.TaskID) (domain.Task, error) {
	var out domain.Task
	err := c.call(ctx, "GetTask", struct {
		TaskID domain.TaskID `json:"taskID"`
	}{id}, &out)
	return out, err
}

func (c *Client) GetMessage(ctx context.Context, id domain.MessageID) (domain.Message, error) {
	var out domain.Message
	err := c.call(ctx, "GetMessage", struct {
		MessageID domain.MessageID `json:"messageID"`
	}{id}, &out)
	return out, err
}

func (c *Client) GetRun(ctx context.Context, id domain.RunID) (domain.Run, error) {
	var out domain.Run
	err := c.call(ctx, "GetRun", struct {
		RunID domain.RunID `json:"runID"`
	}{id}, &out)
	return out, err
}

func (c *Client) ReadOutput(ctx context.Context, id domain.RunID, afterOffset uint64, byteLimit int) (domain.RunOutput, error) {
	var out domain.RunOutput
	err := c.call(ctx, "ReadOutput", struct {
		RunID       domain.RunID `json:"runID"`
		AfterOffset uint64       `json:"afterOffset"`
		ByteLimit   int          `json:"byteLimit"`
	}{id, afterOffset, byteLimit}, &out)
	return out, err
}

func (c *Client) GetOperation(ctx context.Context, id domain.OperationID) (domain.Operation, error) {
	var out domain.Operation
	err := c.call(ctx, "GetOperation", struct {
		OperationID domain.OperationID `json:"operationID"`
	}{id}, &out)
	return out, err
}

func (c *Client) GetSnapshot(ctx context.Context) (domain.Snapshot, error) {
	var out domain.Snapshot
	err := c.call(ctx, "GetSnapshot", struct{}{}, &out)
	return out, err
}

func (c *Client) ResolveRequest(ctx context.Context, id domain.RequestID) (domain.Receipt, error) {
	var out domain.Receipt
	err := c.call(ctx, "ResolveRequest", struct {
		RequestID domain.RequestID `json:"requestID"`
	}{id}, &out)
	return out, err
}

// Subscribe over this file-based request/response transport is a
// single bounded batch, not a live stream: it asks the host's real
// Subscribe (H101-61's event bus, running in-process on the serve
// side) for whatever it can deliver within its own short drain window
// (dispatch.go's subscribeDrainWindow), then returns those events on
// an already-closed channel. This is enough for a fresh reader to
// prove the shipped state-event path — item 5's I6 (H101-193) — but a
// caller wanting continuous delivery must call Subscribe again with a
// later cursor; there is no live push here, and pretending otherwise
// would be exactly the kind of manufactured certainty H101-195
// forbids.
func (c *Client) Subscribe(ctx context.Context, filter string) (<-chan domain.Event, error) {
	events := make([]domain.Event, 0)
	if err := c.call(ctx, "Subscribe", struct {
		AfterCursor string `json:"afterCursor"`
	}{filter}, &events); err != nil {
		return nil, err
	}
	ch := make(chan domain.Event, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}
