package adaptercontract_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-05: provenance, delivery facts (including absent), disclosure, no validation badges.
func TestUI05_ProvenanceDeliveryFactsAndDisclosure(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			registerAgent(t, ctx, driver, env, expected.EngineerID, "ui05-r1", "Engineer")
			registerAgent(t, ctx, driver, env, expected.AnalystID, "ui05-r2", "Analyst")
			createTask(t, ctx, driver, env, "ui05-r3", expected.TaskID, "Investigate", expected.EngineerID)
			sendMessage(t, ctx, driver, env, "ui05-r4", expected.MessageID, expected.AnalystID, "Please investigate", expected.TaskID)

			switch driver.Name() {
			case "cli":
				cli := driver.(*cliDriver)
				pending := queryCLIMessages(ctx, cli, env, expected.AnalystID)
				if pending.ExitCode != 0 {
					t.Fatalf("messages failed: %q", pending.Stderr)
				}
				assertDisclosure(t, pending.Stderr)
				assertMessageDeliveryFacts(t, pending.Stdout, false)
				assertNoValidationBadge(t, pending.Stdout)
				if !strings.Contains(pending.Stdout, "claimed") {
					t.Fatalf("sender must be shown as claimed routing data: %s", pending.Stdout)
				}

				detail := queryCLIMessage(ctx, cli, env, expected.MessageID)
				if detail.ExitCode != 0 {
					t.Fatalf("message failed: %q", detail.Stderr)
				}
				assertDisclosure(t, detail.Stderr)
				assertMessageDeliveryFacts(t, detail.Stdout, false)

				ackMessage(t, ctx, driver, env, "ui05-r5", expected.MessageID, expected.AnalystID)
				acked := queryCLIMessage(ctx, cli, env, expected.MessageID)
				assertMessageDeliveryFacts(t, acked.Stdout, true)
			case "throwaway":
				td := driver.(*throwawayDriver)
				snap, result := throwawaySnapshot(t, ctx, td, env, domain.AgentID(expected.AnalystID))
				if result.ExitCode != 0 {
					t.Fatalf("snapshot failed: %q", result.Stderr)
				}
				if len(snap.Messages) == 0 {
					t.Fatal("snapshot discovery found no messages without prior message ID")
				}
				msg, result := throwawayMessage(t, ctx, td, env, domain.AgentID(expected.AnalystID), domain.MessageID(expected.MessageID))
				if result.ExitCode != 0 {
					t.Fatalf("message query failed: %q", result.Stderr)
				}
				if msg.AcknowledgedAt != nil {
					t.Fatal("message should be unacknowledged before ack")
				}
				for _, fact := range []*time.Time{msg.QueuedAt, msg.PublishedAt, msg.ProcessedAt} {
					if fact == nil {
						t.Fatal("delivery fact timestamp missing on throwaway message view")
					}
				}
				if msg.Provenance.IdentityVerification != domain.IdentityUnverified {
					t.Fatalf("provenance verification = %q, want unverified", msg.Provenance.IdentityVerification)
				}
				ackMessage(t, ctx, driver, env, "ui05-r5", expected.MessageID, expected.AnalystID)
				acked, _ := throwawayMessage(t, ctx, td, env, domain.AgentID(expected.AnalystID), domain.MessageID(expected.MessageID))
				if acked.AcknowledgedAt == nil {
					t.Fatal("acknowledged message missing AcknowledgedAt")
				}
			}
		})
	}
}
