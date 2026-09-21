// Package forbidden is intentionally outside the normal package graph. CI
// runs the import checker against it and requires the checker to fail.
package forbidden

import (
	"context"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/statestore"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/ports"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
)

var _ = task.CallerScope{}

func reachPersistence(root string) error {
	store, err := statestore.Open(root)
	if err != nil {
		return err
	}
	defer store.Close()
	_, _, err = store.Commit(context.Background(), domain.WorkspaceID("fixture"), ports.CommitRequest{
		CallerAgentID:      domain.AgentID("fixture"),
		RequestID:          domain.RequestID("fixture"),
		PayloadFingerprint: "fixture",
		Mutate:             func(*domain.Snapshot) ([]domain.Event, error) { return nil, nil },
	})
	return err
}
