package agent

// EventType identifies the kind of agent-loop observation emitted through
// Options.OnEvent (ADR-00006). Emitting events is strictly optional and must
// not change loop behavior.
type EventType string

const (
	// EventStart marks the start of Run for one prompt.
	EventStart EventType = "start"
	// EventText carries a non-terminal model text fragment from an iteration.
	EventText EventType = "text"
	// EventTool marks a tool call lifecycle: "start" / "result" in Event.Phase.
	EventTool EventType = "tool"
	// EventFinal carries the terminal model answer text.
	EventFinal EventType = "final"
)

// ToolPhase brackets one tool execution.
type ToolPhase string

const (
	ToolPhaseStart  ToolPhase = "start"
	ToolPhaseResult ToolPhase = "result"
)

// Event is a single observation. All fields are safe to log (no secrets).
type Event struct {
	Type     EventType
	Text     string
	ToolName string
	ToolArgs map[string]any
	Phase    ToolPhase
	Bytes    int
}