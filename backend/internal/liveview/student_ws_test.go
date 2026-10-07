package liveview

import (
	"testing"
)

// containsVideoProctoring decides whether an assignment's integrity policy
// enables video proctoring. It runs on every student WS connect, so a false
// negative silently disables the feature.

func TestContainsVideoProctoring(t *testing.T) {
	cases := []struct {
		name   string
		policy string
		want   bool
	}{
		{"enabled compact", `{"video_proctoring":true}`, true},
		{"enabled spaced", `{"video_proctoring": true}`, true},
		{"disabled", `{"video_proctoring":false}`, false},
		{"key absent", `{"webcam":true}`, false},
		{"empty object", `{}`, false},
		{"empty slice", ``, false},
		{"enabled among others", `{"webcam":false,"video_proctoring":true,"screen":true}`, true},
		{"disabled among others", `{"webcam":true,"video_proctoring":false}`, false},
		{"null value", `{"video_proctoring":null}`, false},
		// String-valued flags always read as false. json.Unmarshal into a bool
		// fails for a JSON string, and the fallback compares the raw bytes
		// (which still carry their quotes) against `true`, so it cannot match.
		// Pinned as-is: see TestContainsVideoProctoringStringValueQuirk.
		{"string true reads false", `{"video_proctoring":"true"}`, false},
		{"string false", `{"video_proctoring":"false"}`, false},
		{"string other", `{"video_proctoring":"yes"}`, false},
		{"string with spaces", `{"video_proctoring":"  true  "}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := containsVideoProctoring([]byte(c.policy)); got != c.want {
				t.Fatalf("containsVideoProctoring(%s)=%v, want %v", c.policy, got, c.want)
			}
		})
	}
}

// TestContainsVideoProctoringNonJSONFallback pins the fallback branch: when the
// policy is not valid JSON, the code substring-matches the raw bytes. This
// matches the shape Postgres returns for a jsonb column read as text.
func TestContainsVideoProctoringNonJSONFallback(t *testing.T) {
	if !containsVideoProctoring([]byte(`not json "video_proctoring":true`)) {
		t.Fatal("substring fallback must still detect an enabled flag")
	}
	if containsVideoProctoring([]byte(`not json "video_proctoring":false`)) {
		t.Fatal("substring fallback must not report false as enabled")
	}
}

func TestContainsVideoProctoringMalformedValueTypes(t *testing.T) {
	// A nested object where a bool is expected must not be treated as enabled.
	if containsVideoProctoring([]byte(`{"video_proctoring":{"on":true}}`)) {
		t.Fatal("object value must not count as enabled")
	}
}

// TestContainsVideoProctoringStringValueQuirk records a latent defect: the
// non-bool fallback compares the raw JSON bytes against "true", but a JSON
// string value still carries its quotes, so a policy written as
// {"video_proctoring":"true"} is treated as DISABLED and proctoring silently
// never starts. Today the API writes a real boolean, so this is latent rather
// than live — but it is a trap for any future writer that emits a string.
func TestContainsVideoProctoringStringValueQuirk(t *testing.T) {
	got := containsVideoProctoring([]byte(`{"video_proctoring":"true"}`))
	if got {
		t.Skip("string-valued true is now honoured; the quirk has been fixed")
	}
	t.Log("LATENT: containsVideoProctoring treats a JSON string \"true\" as disabled " +
		"because the fallback compares quoted bytes; only a real boolean enables proctoring")
}
