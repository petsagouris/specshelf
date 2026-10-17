package cli

import (
	"strings"
	"testing"
)

// TestSplitValues pins the rule the CLI spec states: comma-separated parts and
// repeated flags alike, each trimmed, empty parts dropped, de-duplicated while
// preserving first-seen order.
func TestSplitValues(t *testing.T) {
	tests := []struct {
		name   string
		groups [][]string
		want   string
	}{
		{name: "no values", groups: nil, want: ""},
		{name: "repeated flags", groups: [][]string{{"a"}, {"b"}}, want: "a,b"},
		{name: "comma list", groups: [][]string{{"a,b,c"}}, want: "a,b,c"},
		{name: "surrounding whitespace trimmed", groups: [][]string{{" a ,\tb\n"}}, want: "a,b"},
		{name: "empty parts dropped", groups: [][]string{{"a,,b", "", ","}}, want: "a,b"},
		{name: "de-duplicated keeping first-seen order", groups: [][]string{{"b,a"}, {"a", "c", "b"}}, want: "b,a,c"},
		{name: "aliases interleave in the order given", groups: [][]string{{"a"}, {"c"}, {"b"}}, want: "a,c,b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitValues(tt.groups...); strings.Join(got, ",") != tt.want {
				t.Errorf("splitValues(%v) = %q, want %q", tt.groups, strings.Join(got, ","), tt.want)
			}
		})
	}
}

// TestSessionValues pins the current-session fallback: CODEX_THREAD_ID, then
// CODEX_SESSION_ID, and an error naming the variable when neither is set.
func TestSessionValues(t *testing.T) {
	t.Run("current flag off passes the list through", func(t *testing.T) {
		t.Setenv("CODEX_THREAD_ID", "ignored")
		got, err := sessionValues([]string{" s1 ,s2", "s2"}, false)
		if err != nil {
			t.Fatalf("sessionValues: %v", err)
		}
		if strings.Join(got, ",") != "s1,s2" {
			t.Errorf("sessionValues = %v, want [s1 s2]", got)
		}
	})

	t.Run("CODEX_THREAD_ID wins over CODEX_SESSION_ID", func(t *testing.T) {
		t.Setenv("CODEX_THREAD_ID", "thread-1")
		t.Setenv("CODEX_SESSION_ID", "session-1")
		got, err := sessionValues(nil, true)
		if err != nil {
			t.Fatalf("sessionValues: %v", err)
		}
		if strings.Join(got, ",") != "thread-1" {
			t.Errorf("sessionValues = %v, want [thread-1]", got)
		}
	})

	t.Run("falls back to CODEX_SESSION_ID", func(t *testing.T) {
		t.Setenv("CODEX_THREAD_ID", "")
		t.Setenv("CODEX_SESSION_ID", "session-2")
		got, err := sessionValues(nil, true)
		if err != nil {
			t.Fatalf("sessionValues: %v", err)
		}
		if strings.Join(got, ",") != "session-2" {
			t.Errorf("sessionValues = %v, want [session-2]", got)
		}
	})

	t.Run("the current id is appended and de-duplicated", func(t *testing.T) {
		t.Setenv("CODEX_THREAD_ID", "s2")
		got, err := sessionValues([]string{"s1", "s2"}, true)
		if err != nil {
			t.Fatalf("sessionValues: %v", err)
		}
		if strings.Join(got, ",") != "s1,s2" {
			t.Errorf("sessionValues = %v, want [s1 s2]", got)
		}
	})

	t.Run("neither variable set is an error naming the variable", func(t *testing.T) {
		t.Setenv("CODEX_THREAD_ID", "")
		t.Setenv("CODEX_SESSION_ID", "")
		got, err := sessionValues(nil, true)
		if err == nil {
			t.Fatalf("sessionValues succeeded with neither variable set, got %v", got)
		}
		if !strings.Contains(err.Error(), "CODEX_THREAD_ID") {
			t.Errorf("error %q does not name the missing variable", err)
		}
	})
}
