package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDiagnosticPayloadV2JSONRoundTrip(t *testing.T) {
	in := DiagnosticPayloadV2{
		Version: "v2",
		OverallSQI: 72.5,
		Dimensions: SQIDimensionsV2{Mastery: 0.8, Speed: 0.6, Risk: 0.2, Coverage: 0.9},
		ExamSummary: ExamSummaryV2{
			ExamType: "diagnostic", TotalQuestions: 60, Attempted: 55, Correct: 40, Wrong: 10, Skipped: 4, Unseen: 6,
		},
		ConceptProfiles: []ConceptProfileV2{
			{ConceptTag: "algebra", Subject: "math", Status: StatusConfusedV2, PriorityRank: 1},
		},
		BehaviorFlags: BehaviorFlagsV2{
			PanicGuesser: BehaviorFlagV2{Detected: true, Confidence: 0.7, Evidence: "3 wrong in <10s"},
		},
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out DiagnosticPayloadV2
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Version != "v2" || out.OverallSQI != 72.5 || out.Dimensions.Mastery != 0.8 {
		t.Fatalf("round trip mismatch: %+v", out)
	}
	if out.ExamSummary.TotalQuestions != 60 || out.ConceptProfiles[0].Status != StatusConfusedV2 {
		t.Fatalf("round trip mismatch: %+v", out)
	}
}

func TestDiagnosticPayloadV2JSONFieldNames(t *testing.T) {
	data, _ := json.Marshal(DiagnosticPayloadV2{})
	s := string(data)
	for _, key := range []string{`"overall_sqi"`, `"dimensions"`, `"exam_summary"`, `"attempt_profile"`, `"concept_profiles"`, `"behavior_flags"`, `"first_half_accuracy"`, `"second_half_accuracy"`} {
		if !strings.Contains(s, key) {
			t.Errorf("missing JSON key %s in %s", key, s)
		}
	}
}

func TestConceptStatusConstants(t *testing.T) {
	if StatusMasteredV2 != "mastered" || StatusAlmostThereV2 != "almost_there" ||
		StatusConfusedV2 != "confused" || StatusNotStudiedV2 != "not_studied" || StatusNotReachedV2 != "not_reached" {
		t.Fatal("ConceptStatusV2 constants changed; this is a wire-format contract")
	}
}
