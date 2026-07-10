package cli

import (
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestParseSkipStepsRejectsReview(t *testing.T) {
	if _, err := parseSkipSteps("review"); err == nil {
		t.Fatal("expected --skip=review to be rejected, got nil error")
	}
	// Even mixed in with a skippable step, review must poison the whole list.
	if _, err := parseSkipSteps("lint,review"); err == nil {
		t.Fatal("expected --skip=lint,review to be rejected, got nil error")
	}
}

func TestParseSkipStepsAllowsSkippable(t *testing.T) {
	steps, err := parseSkipSteps("lint,test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("want 2 steps, got %v", steps)
	}
}

func TestParseSkipPushOptionsRejectsReview(t *testing.T) {
	_, err := parseSkipPushOptions([]string{"no-mistakes.skip=review"})
	if err == nil {
		t.Fatal("expected push-option skip=review to be rejected")
	}
}

func TestIsMandatoryStep(t *testing.T) {
	if !types.IsMandatoryStep(types.StepReview) {
		t.Error("review should be mandatory")
	}
	for _, s := range []types.StepName{types.StepLint, types.StepTest, types.StepDocument, types.StepCI} {
		if types.IsMandatoryStep(s) {
			t.Errorf("%q should be skippable", s)
		}
	}
	// Error message should name the review gate so agents get a clear reason.
	_, err := parseSkipSteps("review")
	if err == nil || !strings.Contains(err.Error(), "review") {
		t.Errorf("skip error should mention review, got %v", err)
	}
}
