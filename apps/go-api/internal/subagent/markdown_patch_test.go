package subagent

import (
	"strings"
	"testing"
)

func TestPatchSection_ReplacesBodyAndPreservesSiblings(t *testing.T) {
	doc := "# Title\n\n" +
		"### Q1. Compensation\n_Answer:_ (pending)\n\n" +
		"### Q2. Work authorization\n_Answer:_ (pending)\n\n" +
		"### Q3. Notice period\n_Answer:_ (pending)\n"

	got, err := PatchSection(doc, "Q1. Compensation", "_Answer:_ $500k base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "### Q1. Compensation\n_Answer:_ $500k base\n") {
		t.Errorf("Q1 body not replaced; got:\n%s", got)
	}
	if !strings.Contains(got, "### Q2. Work authorization\n_Answer:_ (pending)") {
		t.Errorf("Q2 sibling not preserved; got:\n%s", got)
	}
	if !strings.Contains(got, "### Q3. Notice period\n_Answer:_ (pending)") {
		t.Errorf("Q3 sibling not preserved; got:\n%s", got)
	}
	if strings.Count(got, "Q1. Compensation") != 1 {
		t.Errorf("expected exactly one Q1. Compensation heading, got:\n%s", got)
	}
}

func TestPatchSection_UnknownSectionReturnsError(t *testing.T) {
	doc := "# Title\n\n## 5. Logistics\nbody\n"
	got, err := PatchSection(doc, "Compensation", "anything")
	if err == nil {
		t.Fatalf("expected error for unknown section, got doc:\n%s", got)
	}
	if got != "" {
		t.Errorf("expected empty doc on error, got:\n%s", got)
	}
}

func TestPatchSection_CaseInsensitiveMatch(t *testing.T) {
	doc := "## 5. Logistics\noriginal\n"
	got, err := PatchSection(doc, "5. logistics", "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "## 5. Logistics\nupdated") {
		t.Errorf("case-insensitive match failed; got:\n%s", got)
	}
}
