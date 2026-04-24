package workspace

import "testing"

func TestName(t *testing.T) {
	const id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	cases := []struct {
		sessionName string
		want        string
	}{
		{"fix-login", "fix-login-a1b2c3d4"},
		{"Fix Login!", "fix-login-a1b2c3d4"},
		{"  spaces  ", "spaces-a1b2c3d4"},
		{"___", "a1b2c3d4"},
		{"", "a1b2c3d4"},
		{"a-very-long-session-name-that-should-be-truncated", "a-very-long-session-name-a1b2c3d4"},
		{"Émoji 🔥 test", "moji-test-a1b2c3d4"},
	}
	for _, tc := range cases {
		if got := Name(tc.sessionName, id); got != tc.want {
			t.Errorf("Name(%q) = %q, want %q", tc.sessionName, got, tc.want)
		}
	}
}

func TestNameShortID(t *testing.T) {
	// Short UUIDs shouldn't blow up
	if got := Name("", "abc"); got != "abc" {
		t.Errorf("short id fallthrough: got %q", got)
	}
}
