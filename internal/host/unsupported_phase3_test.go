package host

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

func TestFrontendSession_Phase3OperationsAreUnsupported(t *testing.T) {
	// StartRun moved off this list in H101-144 (Phase 3 item 1): it is a
	// real command now, not an Unsupported stub — see
	// TestFrontendSession_StartRun* in start_run_test.go. StopRun and
	// SetRunBudget remain items 2/3's scope.
	session := &frontendSession{}
	tests := []struct {
		name string
		call func() error
	}{
		{name: "StopRun", call: func() error { _, err := session.StopRun(context.Background(), api.StopRunRequest{}); return err }},
		{name: "SetRunBudget", call: func() error {
			_, err := session.SetRunBudget(context.Background(), api.SetRunBudgetRequest{})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			derr, ok := err.(*domain.Error)
			if !ok || derr.Code != domain.ErrUnsupported {
				t.Fatalf("error = %T %v, want domain.Unsupported", err, err)
			}
			if derr.Detail == "" {
				t.Fatal("unsupported error must explain that the operation is not implemented")
			}
		})
	}
}
