package main

import (
	"strings"
	"testing"
)

func TestNotificationInput(t *testing.T) {
	cases := []struct {
		name  string
		input appointment
		valid bool
	}{
		{"confirmation", appointment{"visit-42", "2026-10-02T09:00:00Z", "confirmed"}, true},
		{"cancelled", appointment{"visit-42", "2026-10-02T09:00:00Z", "cancelled"}, true},
		{"clinical state", appointment{"visit-42", "2026-10-02T09:00:00Z", "diagnosis"}, false},
		{"invalid time", appointment{"visit-42", "tomorrow", "confirmed"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prompt, err := notificationInput(tc.input)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, err=%v", tc.valid, err)
			}
			if tc.valid && (!strings.Contains(prompt, "no diagnosis") || !strings.Contains(prompt, tc.input.Reference)) {
				t.Fatalf("unsafe prompt: %q", prompt)
			}
		})
	}
}
