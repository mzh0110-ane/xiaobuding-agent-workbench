package reliable

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// @covers AC-RELIABLE-RENEWAL-001.4
func TestExhaustedParentCancellationCannotInterruptSeedOwnership(t *testing.T) {
	for _, reserved := range []bool{false, true} {
		name := "idle-parent"
		if reserved {
			name = "reserved-for-seed"
		}
		t.Run(name, func(t *testing.T) {
			e, parent := preparedExhaustedTask(t, &fixtureAgent{})
			claimExhaustedChild(t, e, parent, "successor")
			ctx := context.Background()
			if reserved {
				var release func()
				var err error
				ctx, release, err = e.reserve(ctx, parent.ID)
				if err != nil {
					t.Fatal(err)
				}
				defer release()
			}
			err := e.Cancel(parent.ID)
			if err == nil || !strings.Contains(err.Error(), "batch_successor_exists") {
				t.Fatalf("superseded parent cancellation was admitted: %v", err)
			}
			current, loadErr := e.Store.Get(parent.ID)
			if loadErr != nil || current.State != stateExhausted || current.Round != MaxRounds || ctx.Err() != nil {
				t.Fatalf("old batch or seed ownership changed: state=%s round=%d context=%v load=%v", current.State, current.Round, ctx.Err(), loadErr)
			}
		})
	}
}

// @covers AC-RELIABLE-RENEWAL-001.4
func TestExhaustedParentQuestionStillCancelsWithoutStoppingChild(t *testing.T) {
	e, parent := preparedExhaustedTask(t, &fixtureAgent{})
	child := claimExhaustedChild(t, e, parent, "question-successor")
	_, err := e.Store.Update(parent.ID, func(saved *Task) error {
		saved.Messages = append(saved.Messages, TaskMessage{ID: "question", Kind: messageQuestion, Content: "Explain the previous batch", Status: messageReading})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, release, err := e.reserve(context.Background(), parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	e.mu.Lock()
	e.questions = map[string]bool{parent.ID: true}
	e.mu.Unlock()
	if err = e.Cancel(parent.ID); err != nil {
		t.Fatalf("actual read-only answer could not be stopped: %v", err)
	}
	current, err := e.Store.Get(parent.ID)
	if err != nil || ctx.Err() == nil || current.State != stateExhausted || current.Round != MaxRounds || current.Messages[len(current.Messages)-1].Status != stateCancelled {
		t.Fatalf("question stop changed old work: %+v context=%v error=%v", current, ctx.Err(), err)
	}
	successor, err := e.Store.Get(child.ID)
	if err != nil || successor.State != stateQueued || successor.Round != 0 {
		t.Fatalf("parent question stop changed the successor: %+v %v", successor, err)
	}
}

// @covers AC-RELIABLE-RENEWAL-001.5
func TestExhaustedParentFutureSettingsRedirectToSuccessor(t *testing.T) {
	e, parent := preparedExhaustedTask(t, &fixtureAgent{})
	plan := e.defaultPlan()
	before, err := e.UpdateFutureModelPlan(parent.ID, 0, plan)
	if err != nil {
		t.Fatalf("valid future settings positive control failed: %v", err)
	}
	child := claimExhaustedChild(t, e, before, "settings-successor")
	_, err = e.UpdateFutureModelPlan(parent.ID, before.FuturePlan.Revision, plan)
	if err == nil || !strings.Contains(err.Error(), "batch_successor_exists") {
		t.Fatalf("old batch silently accepted settings for an existing successor: %v", err)
	}
	current, err := e.Store.Get(parent.ID)
	if err != nil || !reflect.DeepEqual(current.FuturePlan, before.FuturePlan) {
		t.Fatalf("old batch future preferences changed: %+v %v", current.FuturePlan, err)
	}
	successor, err := e.Store.Get(child.ID)
	if err != nil || successor.ModelPlan != plan {
		t.Fatalf("existing child assignment changed: %+v %v", successor.ModelPlan, err)
	}
}
