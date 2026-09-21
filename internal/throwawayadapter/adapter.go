// Package throwawayadapter is an intentionally disposable, machine-facing
// frontend. It accepts one structured JSON operation at a time, unlike the
// command-line adapter's flags and exit statuses. It depends only on the
// neutral api contract and passive domain records.
package throwawayadapter

import (
	"context"
	"encoding/json"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

type Adapter struct{ session api.FrontendSession }

func New(session api.FrontendSession) *Adapter { return &Adapter{session: session} }

type Envelope struct {
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload"`
}

type Response struct {
	OK     bool        `json:"ok"`
	Result interface{} `json:"result,omitempty"`
	Code   string      `json:"code,omitempty"`
	Detail string      `json:"detail,omitempty"`
}

func (a *Adapter) Handle(ctx context.Context, input []byte) ([]byte, error) {
	var envelope Envelope
	if err := json.Unmarshal(input, &envelope); err != nil {
		return json.Marshal(Response{Code: string(domain.ErrInvalidArgument), Detail: "invalid operation envelope"})
	}
	result, err := a.dispatch(ctx, envelope)
	if err != nil {
		return json.Marshal(errorResponse(err))
	}
	return json.Marshal(Response{OK: true, Result: result})
}

func errorResponse(err error) Response {
	var derr *domain.Error
	if !asDomainError(err, &derr) {
		return Response{Code: string(domain.ErrIOFailure), Detail: err.Error()}
	}
	return Response{Code: string(derr.Code), Detail: derr.Detail}
}

// Kept local so the adapter's error path has no dependency on CLI rendering.
func asDomainError(err error, target **domain.Error) bool {
	if derr, ok := err.(*domain.Error); ok {
		*target = derr
		return true
	}
	return false
}

func (a *Adapter) dispatch(ctx context.Context, envelope Envelope) (interface{}, error) {
	decode := func(dst interface{}) error { return json.Unmarshal(envelope.Payload, dst) }
	switch envelope.Operation {
	case "register":
		var req api.RegisterAgentRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.RegisterAgent(ctx, req)
	case "update":
		var req api.UpdateAgentRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.UpdateAgent(ctx, req)
	case "create":
		var req api.CreateTaskRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.CreateTask(ctx, req)
	case "transition":
		var req api.TransitionTaskRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.TransitionTask(ctx, req)
	case "report":
		var req api.ReportTaskResultRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.ReportTaskResult(ctx, req)
	case "accept":
		var req api.AcceptTaskResultRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.AcceptTaskResult(ctx, req)
	case "reject":
		var req api.RejectTaskResultRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.RejectTaskResult(ctx, req)
	case "send":
		var req api.SendMessageRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.SendMessage(ctx, req)
	case "ack":
		var req api.AcknowledgeMessageRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.AcknowledgeMessage(ctx, req)
	case "start-run":
		var req api.StartRunRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.StartRun(ctx, req)
	case "stop-run":
		var req api.StopRunRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.StopRun(ctx, req)
	case "set-run-budget":
		var req api.SetRunBudgetRequest
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.SetRunBudget(ctx, req)
	case "agent":
		var req struct {
			AgentID domain.AgentID `json:"agentID"`
		}
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.GetAgent(ctx, req.AgentID)
	case "task":
		var req struct {
			TaskID domain.TaskID `json:"taskID"`
		}
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.GetTask(ctx, req.TaskID)
	case "message":
		var req struct {
			MessageID domain.MessageID `json:"messageID"`
		}
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.GetMessage(ctx, req.MessageID)
	case "snapshot":
		return a.session.GetSnapshot(ctx)
	case "resolve":
		var req struct {
			RequestID domain.RequestID `json:"requestID"`
		}
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		return a.session.ResolveRequest(ctx, req.RequestID)
	case "subscribe":
		var req struct {
			AfterCursor string `json:"afterCursor"`
		}
		if err := decode(&req); err != nil {
			return nil, invalidPayload()
		}
		stream, err := a.session.Subscribe(ctx, req.AfterCursor)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"subscribed": true, "events": stream != nil}, nil
	default:
		return nil, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "unknown operation"}
	}
}

func invalidPayload() error {
	return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "invalid operation payload"}
}
