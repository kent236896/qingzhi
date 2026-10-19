package knowledge

import (
	"testing"
	"time"
)

func TestComputeWikiStatus(t *testing.T) {
	timeout := 30 * time.Minute
	cases := []struct {
		name  string
		input WikiStatusInput
		want  string
	}{
		{name: "disabled", input: WikiStatusInput{}, want: WikiDisabled},
		{name: "empty", input: WikiStatusInput{Enabled: true}, want: WikiEmpty},
		{name: "parsing", input: WikiStatusInput{Enabled: true, DocTotal: 2, DocTerminal: 1}, want: WikiGenerating},
		{name: "active", input: WikiStatusInput{Enabled: true, DocTotal: 1, DocTerminal: 1, IsActive: true, TotalPages: 3}, want: WikiGenerating},
		{name: "pending", input: WikiStatusInput{Enabled: true, DocTotal: 1, DocTerminal: 1, PendingTasks: 2}, want: WikiGenerating},
		{name: "ready", input: WikiStatusInput{Enabled: true, DocTotal: 1, DocTerminal: 1, TotalPages: 4}, want: WikiReady},
		{name: "waiting", input: WikiStatusInput{Enabled: true, DocTotal: 1, DocTerminal: 1, Elapsed: time.Minute, Timeout: timeout}, want: WikiGenerating},
		{name: "timeout", input: WikiStatusInput{Enabled: true, DocTotal: 1, DocTerminal: 1, Elapsed: timeout, Timeout: timeout}, want: WikiTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComputeWikiStatus(tc.input); got != tc.want {
				t.Fatalf("status = %s, want %s", got, tc.want)
			}
		})
	}
}
