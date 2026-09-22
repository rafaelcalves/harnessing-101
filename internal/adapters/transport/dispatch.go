package transport

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// subscribeDrainWindow bounds how long the "Subscribe" operation below
// waits for events before answering — see its own doc comment for why
// this is a single bounded batch, not a live stream, over this
// request/response transport.
const subscribeDrainWindow = 200 * time.Millisecond

// Dispatch decodes one Envelope's Payload against the api.FrontendSession
// method its Operation names, calls it, and re-encodes the result or
// error as a Response. It is the host side's whole authority surface:
// every case here is a straight forward to an already-existing,
// already-tested method — this function adds no policy of its own.
func Dispatch(ctx context.Context, session api.FrontendSession, envelope Envelope) Response {
	result, err := dispatch(ctx, session, envelope)
	if err != nil {
		return Response{Error: errorInfo(err)}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return Response{Error: &ErrorInfo{Code: string(domain.ErrIOFailure), Detail: err.Error()}}
	}
	return Response{OK: true, Result: encoded}
}

func dispatch(ctx context.Context, session api.FrontendSession, envelope Envelope) (interface{}, error) {
	decode := func(dst interface{}) error {
		if err := json.Unmarshal(envelope.Payload, dst); err != nil {
			return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "invalid operation payload"}
		}
		return nil
	}
	switch envelope.Operation {
	case "RegisterAgent":
		var req api.RegisterAgentRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.RegisterAgent(ctx, req)
	case "UpdateAgent":
		var req api.UpdateAgentRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.UpdateAgent(ctx, req)
	case "CreateTask":
		var req api.CreateTaskRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.CreateTask(ctx, req)
	case "TransitionTask":
		var req api.TransitionTaskRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.TransitionTask(ctx, req)
	case "ReportTaskResult":
		var req api.ReportTaskResultRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.ReportTaskResult(ctx, req)
	case "AcceptTaskResult":
		var req api.AcceptTaskResultRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.AcceptTaskResult(ctx, req)
	case "RejectTaskResult":
		var req api.RejectTaskResultRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.RejectTaskResult(ctx, req)
	case "SendMessage":
		var req api.SendMessageRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.SendMessage(ctx, req)
	case "AcknowledgeMessage":
		var req api.AcknowledgeMessageRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.AcknowledgeMessage(ctx, req)
	case "ApproveProfile":
		var req api.ApproveProfileRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.ApproveProfile(ctx, req)
	case "StartRun":
		var req api.StartRunRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.StartRun(ctx, req)
	case "StopRun":
		var req api.StopRunRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.StopRun(ctx, req)
	case "SetRunBudget":
		var req api.SetRunBudgetRequest
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.SetRunBudget(ctx, req)
	case "GetAgent":
		var req struct {
			AgentID domain.AgentID `json:"agentID"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.GetAgent(ctx, req.AgentID)
	case "GetTask":
		var req struct {
			TaskID domain.TaskID `json:"taskID"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.GetTask(ctx, req.TaskID)
	case "GetMessage":
		var req struct {
			MessageID domain.MessageID `json:"messageID"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.GetMessage(ctx, req.MessageID)
	case "GetRun":
		var req struct {
			RunID domain.RunID `json:"runID"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.GetRun(ctx, req.RunID)
	case "ReadOutput":
		var req struct {
			RunID       domain.RunID `json:"runID"`
			AfterOffset uint64       `json:"afterOffset"`
			ByteLimit   int          `json:"byteLimit"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		reader, ok := session.(interface {
			ReadOutput(context.Context, domain.RunID, uint64, int) (domain.RunOutput, error)
		})
		if !ok {
			return nil, &domain.Error{Code: domain.ErrUnsupported, Detail: "output read is unsupported on this session"}
		}
		return reader.ReadOutput(ctx, req.RunID, req.AfterOffset, req.ByteLimit)
	case "GetOperation":
		var req struct {
			OperationID domain.OperationID `json:"operationID"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.GetOperation(ctx, req.OperationID)
	case "GetSnapshot":
		return session.GetSnapshot(ctx)
	case "Subscribe":
		// H101-193's I6 negative needs a real shipped state-event read
		// path for an attached reader, not a live continuous stream
		// (this file-based request/response transport has no natural
		// carrier for that yet — see Client.Subscribe's own doc
		// comment). This drains whatever the host's real, in-process
		// Subscribe delivers within a short bounded window and returns
		// it as one batch; a caller wanting more polls again with the
		// last event's cursor equivalent (WorkspaceRevision).
		var req struct {
			AfterCursor string `json:"afterCursor"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		drainCtx, cancel := context.WithTimeout(ctx, subscribeDrainWindow)
		defer cancel()
		ch, err := session.Subscribe(drainCtx, req.AfterCursor)
		if err != nil {
			return nil, err
		}
		events := make([]domain.Event, 0)
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return events, nil
				}
				events = append(events, ev)
			case <-drainCtx.Done():
				return events, nil
			}
		}
	case "ResolveRequest":
		var req struct {
			RequestID domain.RequestID `json:"requestID"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.ResolveRequest(ctx, req.RequestID)
	default:
		// attach/detach are handled one level up, before a request ever
		// reaches Dispatch, so they are not cases here.
		return nil, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "unknown or unsupported operation: " + envelope.Operation}
	}
}
