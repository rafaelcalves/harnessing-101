package adaptercontract_test

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/throwawayadapter"
)

type throwawayDriver struct{}

func newThrowawayDriver() *throwawayDriver { return &throwawayDriver{} }

func (d *throwawayDriver) Name() string { return "throwaway" }

func (d *throwawayDriver) Invoke(ctx context.Context, env WorkspaceEnv, call Call) Result {
	var stderr bytes.Buffer
	var out Result
	code := assembly.WithSession(&stderr, env.Root, env.WorkspaceID, env.Reviewers, call.Caller, "contract", func(ctx context.Context, session api.FrontendSession) int {
		adapter := throwawayadapter.New(session)
		raw, err := json.Marshal(struct {
			Operation string          `json:"operation"`
			Payload   json.RawMessage `json:"payload"`
		}{Operation: call.ThrowawayOperation, Payload: call.ThrowawayPayload})
		if err != nil {
			out = Result{ExitCode: 1, Code: string(domain.ErrIOFailure), Detail: err.Error()}
			return 1
		}
		respBytes, err := adapter.Handle(ctx, raw)
		if err != nil {
			out = Result{ExitCode: 1, Code: string(domain.ErrIOFailure), Detail: err.Error()}
			return 1
		}
		var resp throwawayadapter.Response
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			out = Result{ExitCode: 1, Code: string(domain.ErrIOFailure), Detail: err.Error()}
			return 1
		}
		out = Result{
			ExitCode: boolExit(resp.OK),
			OK:       resp.OK,
			Stdout:   string(respBytes),
			Stderr:   stderr.String(),
			Code:     resp.Code,
			Detail:   resp.Detail,
		}
		if resp.OK && resp.Result != nil {
			resultBytes, err := json.Marshal(resp.Result)
			if err == nil {
				out.Receipt = parseThrowawayReceiptSilent(resultBytes)
			}
		}
		return out.ExitCode
	})
	out.ExitCode = code
	if out.Stderr == "" {
		out.Stderr = stderr.String()
	}
	return out
}

func (d *throwawayDriver) QueryTask(ctx context.Context, env WorkspaceEnv, taskID domain.TaskID) Result {
	payload, _ := json.Marshal(map[string]string{"taskID": string(taskID)})
	return d.Invoke(ctx, env, Call{
		ThrowawayOperation: "task",
		ThrowawayPayload:   payload,
	})
}

func boolExit(ok bool) int {
	if ok {
		return 0
	}
	return 1
}

func parseThrowawayReceiptSilent(raw []byte) *domain.Receipt {
	var rec domain.Receipt
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil
	}
	return &rec
}
