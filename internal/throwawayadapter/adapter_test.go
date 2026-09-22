package throwawayadapter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

type fakeSession struct{}

func (fakeSession) RegisterAgent(context.Context, api.RegisterAgentRequest) (domain.Receipt, error) {
	return domain.Receipt{RequestID: "r1"}, nil
}
func (fakeSession) UpdateAgent(context.Context, api.UpdateAgentRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) CreateTask(context.Context, api.CreateTaskRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) TransitionTask(context.Context, api.TransitionTaskRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) ReportTaskResult(context.Context, api.ReportTaskResultRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) AcceptTaskResult(context.Context, api.AcceptTaskResultRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) RejectTaskResult(context.Context, api.RejectTaskResultRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) SendMessage(context.Context, api.SendMessageRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) AcknowledgeMessage(context.Context, api.AcknowledgeMessageRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) ApproveProfile(context.Context, api.ApproveProfileRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) StartRun(context.Context, api.StartRunRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) StopRun(context.Context, api.StopRunRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) SetRunBudget(context.Context, api.SetRunBudgetRequest) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) GetAgent(context.Context, domain.AgentID) (domain.Agent, error) {
	return domain.Agent{}, unsupported()
}
func (fakeSession) GetTask(context.Context, domain.TaskID) (domain.Task, error) {
	return domain.Task{}, unsupported()
}
func (fakeSession) GetMessage(context.Context, domain.MessageID) (domain.Message, error) {
	return domain.Message{}, unsupported()
}
func (fakeSession) GetRun(context.Context, domain.RunID) (domain.Run, error) {
	return domain.Run{}, unsupported()
}
func (fakeSession) GetOperation(context.Context, domain.OperationID) (domain.Operation, error) {
	return domain.Operation{}, unsupported()
}
func (fakeSession) GetSnapshot(context.Context) (domain.Snapshot, error) {
	return domain.Snapshot{}, unsupported()
}
func (fakeSession) ResolveRequest(context.Context, domain.RequestID) (domain.Receipt, error) {
	return domain.Receipt{}, unsupported()
}
func (fakeSession) Subscribe(context.Context, string) (<-chan domain.Event, error) {
	return make(chan domain.Event), nil
}

func unsupported() error { return &domain.Error{Code: domain.ErrUnsupported, Detail: "test"} }

type uncertaintySession struct{ fakeSession }

func (uncertaintySession) RegisterAgent(context.Context, api.RegisterAgentRequest) (domain.Receipt, error) {
	revision := uint64(42)
	return domain.Receipt{}, &domain.Error{Code: domain.ErrOutcomeUncertain, Detail: "durability check failed", RequestID: "r-uncertain", WorkspaceID: "ws-1", Effect: domain.EffectApplied, Confirmation: domain.ConfirmationDurability, ObservedRevision: &revision}
}

type plainFailureSession struct{ fakeSession }

func (plainFailureSession) RegisterAgent(context.Context, api.RegisterAgentRequest) (domain.Receipt, error) {
	return domain.Receipt{}, errors.New("unexpected write response")
}

func TestHandleStructuredOperation(t *testing.T) {
	adapter := New(fakeSession{})
	response, err := adapter.Handle(context.Background(), []byte(`{"operation":"register","payload":{"requestID":"r1"}}`))
	if err != nil {
		t.Fatal(err)
	}
	var got Response
	if err := json.Unmarshal(response, &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Error != nil {
		t.Fatalf("response = %+v", got)
	}
}

func TestHandleAdvertisesUnsupportedPhase3Operations(t *testing.T) {
	adapter := New(fakeSession{})
	for _, operation := range []string{"start-run", "stop-run", "set-run-budget"} {
		response, err := adapter.Handle(context.Background(), []byte(`{"operation":"`+operation+`","payload":{}}`))
		if err != nil {
			t.Fatal(err)
		}
		var got Response
		if err := json.Unmarshal(response, &got); err != nil {
			t.Fatal(err)
		}
		if got.Error == nil || got.Error.Code != string(domain.ErrUnsupported) {
			t.Fatalf("%s response = %+v", operation, got)
		}
	}
}

func TestHandlePreservesStructuredUncertainty(t *testing.T) {
	adapter := New(uncertaintySession{})
	response, err := adapter.Handle(context.Background(), []byte(`{"operation":"register","payload":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	var got Response
	if err := json.Unmarshal(response, &got); err != nil {
		t.Fatal(err)
	}
	if got.Error == nil {
		t.Fatalf("response = %+v", got)
	}
	gotErr := got.Error
	if gotErr.Code != string(domain.ErrOutcomeUncertain) || gotErr.RequestID != "r-uncertain" || gotErr.WorkspaceID != "ws-1" || gotErr.Effect != string(domain.EffectApplied) || gotErr.Confirmation != string(domain.ConfirmationDurability) || gotErr.ObservedRevision == nil || *gotErr.ObservedRevision != 42 || !gotErr.Uncertain {
		t.Fatalf("structured uncertainty lost: %+v", gotErr)
	}
}

func TestHandleMarksUnclassifiedMutationUncertain(t *testing.T) {
	adapter := New(plainFailureSession{})
	response, err := adapter.Handle(context.Background(), []byte(`{"operation":"register","payload":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	var got Response
	if err := json.Unmarshal(response, &got); err != nil {
		t.Fatal(err)
	}
	if got.Error == nil || got.Error.Code != string(domain.ErrIOFailure) || !got.Error.Uncertain || got.Error.Effect != string(domain.EffectUnknown) || got.Error.Confirmation != string(domain.ConfirmationOutcome) {
		t.Fatalf("unclassified mutation was not conservatively uncertain: %+v", got.Error)
	}
}
