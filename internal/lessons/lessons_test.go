package lessons

import "testing"

// Following every hint must complete every lesson.
func TestLessonsSolvableWithHints(t *testing.T) {
	for _, l := range All {
		t.Run(l.ID, func(t *testing.T) {
			sb, err := NewSandbox()
			if err != nil {
				t.Fatal(err)
			}
			defer sb.Remove()
			if l.Setup != nil {
				if err := l.Setup(sb); err != nil {
					t.Fatal(err)
				}
			}
			for i, st := range l.Steps {
				if st.Prepare != nil {
					if err := st.Prepare(sb); err != nil {
						t.Fatalf("step %d prepare: %v", i, err)
					}
				}
				out, _ := sb.Shell("project", st.Hint)
				if !st.Done(sb, Attempt{Line: st.Hint, Output: out}) {
					t.Fatalf("step %d (%s) not done after hint %q:\n%s", i, st.Goal, st.Hint, out)
				}
			}
		})
	}
}

func TestAttemptRan(t *testing.T) {
	cases := map[string]bool{
		"git status":          true,
		"git -c x=y status":   true,
		"echo hi; git status": true,
		"echo git":            false,
		"git statusx":         false,
	}
	for line, want := range cases {
		if got := (Attempt{Line: line}).ran("status"); got != want {
			t.Errorf("ran(%q) = %v, want %v", line, got, want)
		}
	}
}
