package pipeline

import (
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

// TestRespondRejectsSkipOnReview verifies the review gate cannot be skipped via
// a gate "skip" action: the response is rejected and the run stays parked, so
// the caller must respond again with approve or fix.
func TestRespondRejectsSkipOnReview(t *testing.T) {
	e := &Executor{approvalCh: make(chan approvalResponse, 1)}
	e.waiting = true
	e.waitingStep = types.StepReview

	if err := e.RespondWithOverrides(types.StepReview, types.ActionSkip, nil, nil, nil); err == nil {
		t.Fatal("expected skip on review to be rejected")
	}
	if !e.waiting {
		t.Fatal("run should stay parked after a rejected skip")
	}

	// approve is accepted and consumes the gate.
	if err := e.RespondWithOverrides(types.StepReview, types.ActionApprove, nil, nil, nil); err != nil {
		t.Fatalf("approve should be accepted: %v", err)
	}
	if e.waiting {
		t.Fatal("run should no longer be waiting after approve")
	}
}

// TestSetSkippedStepsDropsMandatorySteps verifies the server-side backstop:
// skip lists arriving over IPC (push options, rerun params) cannot skip the
// review gate even though the CLI already filters them client-side.
func TestSetSkippedStepsDropsMandatorySteps(t *testing.T) {
	e := &Executor{}
	e.SetSkippedSteps([]types.StepName{types.StepReview, types.StepLint})
	if e.skips[types.StepReview] {
		t.Fatal("mandatory review step must not be skippable via SetSkippedSteps")
	}
	if !e.skips[types.StepLint] {
		t.Fatal("skippable lint step should remain skipped")
	}
}

// TestRespondAllowsSkipOnSkippableStep confirms the guard is scoped to mandatory
// steps only — a skippable step (lint) can still be skipped at its gate.
func TestRespondAllowsSkipOnSkippableStep(t *testing.T) {
	e := &Executor{approvalCh: make(chan approvalResponse, 1)}
	e.waiting = true
	e.waitingStep = types.StepLint

	if err := e.RespondWithOverrides(types.StepLint, types.ActionSkip, nil, nil, nil); err != nil {
		t.Fatalf("skip on lint should be accepted: %v", err)
	}
}
