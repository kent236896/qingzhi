package knowledge

import "time"

const (
	WikiDisabled   = "disabled"
	WikiEmpty      = "empty"
	WikiGenerating = "generating"
	WikiReady      = "ready"
	WikiTimeout    = "timeout"
)

// WikiStatusInput is the measured state used to decide a wiki label.
// WeKnora 0.8.2 stats include is_active and pending_tasks, so those fields
// are the primary signal. A zero page count after every document is terminal
// becomes timeout once the wait exceeds the configured limit.
type WikiStatusInput struct {
	Enabled      bool
	TotalPages   int
	PendingTasks int
	IsActive     bool
	DocTotal     int
	DocTerminal  int
	Elapsed      time.Duration
	Timeout      time.Duration
}

func ComputeWikiStatus(input WikiStatusInput) string {
	if !input.Enabled {
		return WikiDisabled
	}
	if input.DocTotal == 0 {
		return WikiEmpty
	}
	if input.IsActive || input.PendingTasks > 0 || input.DocTerminal < input.DocTotal {
		return WikiGenerating
	}
	if input.TotalPages > 0 {
		return WikiReady
	}
	if input.Timeout > 0 && input.Elapsed >= input.Timeout {
		return WikiTimeout
	}
	return WikiGenerating
}
