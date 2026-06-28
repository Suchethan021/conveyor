package logsec

import "testing"

func TestMask(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"github token", "cloning ghp_abcdefghijklmnopqrstuv1234", "cloning ***REDACTED***"},
		{"token key=value", "TOKEN=supersecretvalue", "TOKEN=***REDACTED***"},
		{"password colon", "password: hunter2", "password: ***REDACTED***"},
		{"bearer header", "auth header Bearer abcDEF123.tok", "auth header Bearer ***REDACTED***"},
		{"no secret untouched", "stage \"build\" started", "stage \"build\" started"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Mask(c.in); got != c.want {
				t.Errorf("Mask(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
