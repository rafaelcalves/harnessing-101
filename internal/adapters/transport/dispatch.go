package transport

import (
	"context"
	"encoding/json"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

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
	case "GetSnapshot":
		return session.GetSnapshot(ctx)
	case "ResolveRequest":
		var req struct {
			RequestID domain.RequestID `json:"requestID"`
		}
		if err := decode(&req); err != nil {
			return nil, err
		}
		return session.ResolveRequest(ctx, req.RequestID)
	default:
		// Subscribe is deliberately absent: it returns a channel, not a
		// value, and this card does not carry a live event stream over
		// the file transport (see Client.Subscribe). attach/detach are
		// handled one level up, before a request ever reaches Dispatch,
		// so they are not cases here either.
		return nil, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "unknown or unsupported operation: " + envelope.Operation}
	}
}
