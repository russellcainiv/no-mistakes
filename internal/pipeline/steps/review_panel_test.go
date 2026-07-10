package steps

import "testing"

func TestRiskRank(t *testing.T) {
	if riskRank("high") <= riskRank("medium") || riskRank("medium") <= riskRank("low") || riskRank("low") <= riskRank("") {
		t.Fatalf("risk ranking not strictly ordered: high=%d medium=%d low=%d none=%d",
			riskRank("high"), riskRank("medium"), riskRank("low"), riskRank(""))
	}
	if riskRank("HIGH") != riskRank("high") {
		t.Errorf("risk rank should be case-insensitive")
	}
}

func TestFindReviewFinding(t *testing.T) {
	items := []Finding{{File: "a.go", Line: 10, Description: "off-by-one"}}
	// Same file+line+description (case/space-insensitive) is a duplicate.
	if findReviewFinding(items, Finding{File: "a.go", Line: 10, Description: "  OFF-BY-ONE "}) == nil {
		t.Error("expected duplicate to be detected")
	}
	// Different line is not a duplicate.
	if findReviewFinding(items, Finding{File: "a.go", Line: 11, Description: "off-by-one"}) != nil {
		t.Error("different line should not be a duplicate")
	}
}

func TestMergeReviewFindingsUpgradesDuplicateSeverity(t *testing.T) {
	var merged Findings
	mergeReviewFindings(&merged, Findings{Items: []Finding{
		{File: "a.go", Line: 10, Description: "off-by-one", Severity: "info"},
	}})
	mergeReviewFindings(&merged, Findings{Items: []Finding{
		{File: "a.go", Line: 10, Description: "off-by-one", Severity: "error"},
	}})
	// A later reviewer grading the same finding higher upgrades it...
	if len(merged.Items) != 1 || merged.Items[0].Severity != "error" {
		t.Fatalf("want single finding upgraded to error, got %+v", merged.Items)
	}
	// ...and a later lower grade never downgrades.
	mergeReviewFindings(&merged, Findings{Items: []Finding{
		{File: "a.go", Line: 10, Description: "off-by-one", Severity: "warning"},
	}})
	if merged.Items[0].Severity != "error" {
		t.Fatalf("severity downgraded to %q", merged.Items[0].Severity)
	}
}

func TestMergeReviewFindingsUnionsAndDedups(t *testing.T) {
	claude := Findings{
		Items: []Finding{
			{File: "a.go", Line: 10, Description: "off-by-one", Severity: "error", Source: "claude"},
			{File: "b.go", Line: 3, Description: "nil deref", Severity: "error", Source: "claude"},
		},
		Summary:   "claude summary",
		RiskLevel: "medium",
	}
	claudeMM := Findings{
		Items: []Finding{
			{File: "a.go", Line: 10, Description: "off-by-one", Severity: "error", Source: "claude-mm"}, // dup of claude's
			{File: "c.go", Line: 42, Description: "unchecked error", Severity: "warning", Source: "claude-mm"},
		},
		Summary:   "mm summary",
		RiskLevel: "high",
	}

	var merged Findings
	mergeReviewFindings(&merged, claude)
	mergeReviewFindings(&merged, claudeMM)

	if len(merged.Items) != 3 {
		t.Fatalf("want 3 unioned findings, got %d: %+v", len(merged.Items), merged.Items)
	}
	// First non-empty summary wins.
	if merged.Summary != "claude summary" {
		t.Errorf("summary = %q, want claude summary", merged.Summary)
	}
	// Highest risk across reviewers escalates the panel.
	if merged.RiskLevel != "high" {
		t.Errorf("risk = %q, want high", merged.RiskLevel)
	}
}

func TestTagFindingsSource(t *testing.T) {
	f := Findings{Items: []Finding{{Description: "x"}, {Description: "y", Source: "explicit"}}}
	tagFindingsSource(&f, "claude-mm")
	if f.Items[0].Source != "claude-mm" {
		t.Errorf("untagged finding source = %q, want claude-mm", f.Items[0].Source)
	}
	if f.Items[1].Source != "explicit" {
		t.Errorf("pre-attributed finding source overwritten: %q", f.Items[1].Source)
	}
}
