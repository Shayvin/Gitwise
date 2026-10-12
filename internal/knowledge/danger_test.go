package knowledge

import "testing"

func TestAnalyze(t *testing.T) {
	cases := []struct {
		args []string
		want Risk
	}{
		{[]string{"push", "--force"}, RiskHigh},
		{[]string{"git", "push", "-uf", "origin", "main"}, RiskHigh},
		{[]string{"push", "--force-with-lease"}, RiskMedium},
		{[]string{"push", "origin", "+main"}, RiskHigh},
		{[]string{"push"}, RiskNone},
		{[]string{"reset", "--hard", "HEAD~1"}, RiskHigh},
		{[]string{"reset", "--soft", "HEAD~1"}, RiskNone},
		{[]string{"clean", "-fd"}, RiskHigh},
		{[]string{"clean", "-n"}, RiskNone},
		{[]string{"restore", "--staged", "a.txt"}, RiskNone},
		{[]string{"restore", "a.txt"}, RiskHigh},
		{[]string{"checkout", "--", "a.txt"}, RiskHigh},
		{[]string{"checkout", "main"}, RiskNone},
		{[]string{"branch", "-D", "x"}, RiskMedium},
		{[]string{"branch", "-d", "x"}, RiskNone},
		{[]string{"rebase", "main"}, RiskMedium},
		{[]string{"rebase", "--abort"}, RiskNone},
		{[]string{"-C", "/tmp", "reset", "--hard"}, RiskHigh},
		{[]string{"gc", "--prune=now"}, RiskHigh},
		{[]string{"status"}, RiskNone},
	}
	for _, c := range cases {
		if got := MaxRisk(Analyze(c.args)); got != c.want {
			t.Errorf("%v: got %v, want %v", c.args, got, c.want)
		}
	}
}

func TestClosest(t *testing.T) {
	for in, want := range map[string]string{"stauts": "status", "comit": "commit", "pusj": "push", "xyzzyq": ""} {
		if got := Closest(in); got != want {
			t.Errorf("Closest(%q) = %q, want %q", in, got, want)
		}
	}
}
