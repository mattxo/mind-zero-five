package mind

import "testing"

// --- isAlreadyDone tests ---

// TestIsAlreadyDoneCompletionLanguage verifies that each known completion signal
// causes isAlreadyDone to return true.
func TestIsAlreadyDoneCompletionLanguage(t *testing.T) {
	cases := []struct {
		name     string
		response string
	}{
		{"explicit tag", "[ALREADY_DONE]"},
		{"tag in prose", "After analysis: [ALREADY_DONE]"},
		{"already done", "This task is already done."},
		{"already implemented", "The feature is already implemented in pkg/foo/bar.go."},
		{"no changes needed", "No changes needed — the code is correct."},
		{"no changes required", "No changes required for this task."},
		{"already exists", "The function already exists at line 42."},
		{"already present", "The test is already present in the test file."},
		{"mixed case", "The validation is Already Implemented correctly."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !isAlreadyDone(tc.response) {
				t.Errorf("isAlreadyDone(%q) = false, want true", tc.response)
			}
		})
	}
}

// TestIsAlreadyDoneNoFalsePositives verifies that "completed" in ambiguous
// contexts does not trigger auto-complete, which would silently drop real work.
func TestIsAlreadyDoneNoFalsePositives(t *testing.T) {
	cases := []struct {
		name     string
		response string
	}{
		{"needs to be completed", "This refactoring needs to be completed before release."},
		{"not yet completed", "Not yet completed — still needs work."},
		{"was completed", "This was completed in a prior commit."},
		{"standalone word", "completed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isAlreadyDone(tc.response) {
				t.Errorf("isAlreadyDone(%q) = true, want false (false positive)", tc.response)
			}
		})
	}
}

// TestIsAlreadyDoneEmptyResponse verifies that an empty response returns false.
func TestIsAlreadyDoneEmptyResponse(t *testing.T) {
	if isAlreadyDone("") {
		t.Error("isAlreadyDone(\"\") = true, want false")
	}
}

// TestIsAlreadyDoneUnrelatedContent verifies that responses with no completion
// indicators return false.
func TestIsAlreadyDoneUnrelatedContent(t *testing.T) {
	cases := []struct {
		name     string
		response string
	}{
		{"error message", "failed to parse input: unexpected token"},
		{"random text", "the quick brown fox jumps over the lazy dog"},
		{"completed word only", "completion is not a match for completed on its own"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isAlreadyDone(tc.response) {
				t.Errorf("isAlreadyDone(%q) = true, want false", tc.response)
			}
		})
	}
}

// --- parseSubtasks tests ---

// TestParseSubtasksNoTags verifies that a response with no [TASK:...] tags
// returns an empty slice.
func TestParseSubtasksNoTags(t *testing.T) {
	cases := []struct {
		name     string
		response string
	}{
		{"empty", ""},
		{"completion language", "This task is already done."},
		{"prose only", "I analyzed the codebase and found no changes required."},
		{"malformed tag", "[TASK:missing-fields]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSubtasks(tc.response)
			if len(got) != 0 {
				t.Errorf("parseSubtasks(%q): got %d subtasks, want 0", tc.response, len(got))
			}
		})
	}
}

// TestParseSubtasksValidTags verifies that well-formed [TASK:subject|desc|model]
// tags are parsed into SubtaskSpec values.
func TestParseSubtasksValidTags(t *testing.T) {
	response := "[TASK:Add validation to pkg/api/handler.go|Validate input and return 400 on error|sonnet]\n" +
		"[TASK:Add test for validation in pkg/api/handler_test.go|Test invalid input returns 400|haiku]"

	got := parseSubtasks(response)
	if len(got) != 2 {
		t.Fatalf("expected 2 subtasks, got %d", len(got))
	}

	if got[0].Subject != "Add validation to pkg/api/handler.go" {
		t.Errorf("subtask[0].Subject = %q", got[0].Subject)
	}
	if got[0].Description != "Validate input and return 400 on error" {
		t.Errorf("subtask[0].Description = %q", got[0].Description)
	}
	if got[0].Model != "sonnet" {
		t.Errorf("subtask[0].Model = %q, want sonnet", got[0].Model)
	}

	if got[1].Model != "haiku" {
		t.Errorf("subtask[1].Model = %q, want haiku", got[1].Model)
	}
}

// TestParseSubtasksInvalidModelDefaultsSonnet verifies that an unrecognised model
// string is replaced with "sonnet".
func TestParseSubtasksInvalidModelDefaultsSonnet(t *testing.T) {
	response := "[TASK:Some subject|Some description|turbo]"
	got := parseSubtasks(response)
	if len(got) != 1 {
		t.Fatalf("expected 1 subtask, got %d", len(got))
	}
	if got[0].Model != "sonnet" {
		t.Errorf("model = %q, want sonnet (default for unknown model)", got[0].Model)
	}
}
