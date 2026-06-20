package ai

// SSE event names emitted by the agent over the chat stream.
const (
	EventMessage        = "message"         // assistant text delta: {content}
	EventToolCall       = "tool_call"       // a tool is being invoked: {tool, tool_call_id, args}
	EventToolResult     = "tool_result"     // tool finished: {tool, tool_call_id, result, is_error}
	EventActionRequired = "action_required" // mutation awaiting approval: {tool, tool_call_id, args, session_id}
	EventError          = "error"           // {message}
	EventDone           = "done"            // {}
)

// SSEEvent is a single server-sent event with a named type and JSON payload.
type SSEEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// EmitFunc sends a single event to the client. Implementations must be safe to
// call sequentially from the agent loop.
type EmitFunc func(SSEEvent)

func messageEvent(content string) SSEEvent {
	return SSEEvent{Event: EventMessage, Data: map[string]any{"content": content}}
}

func toolCallEvent(tool, callID string, args map[string]any) SSEEvent {
	return SSEEvent{Event: EventToolCall, Data: map[string]any{
		"tool":         tool,
		"tool_call_id": callID,
		"args":         args,
	}}
}

func toolResultEvent(tool, callID, result string, isErr bool) SSEEvent {
	return SSEEvent{Event: EventToolResult, Data: map[string]any{
		"tool":         tool,
		"tool_call_id": callID,
		"result":       result,
		"is_error":     isErr,
	}}
}

func actionRequiredEvent(tool, callID string, args map[string]any, sessionID string) SSEEvent {
	return SSEEvent{Event: EventActionRequired, Data: map[string]any{
		"tool":         tool,
		"tool_call_id": callID,
		"args":         args,
		"session_id":   sessionID,
	}}
}

func errorEvent(msg string) SSEEvent {
	return SSEEvent{Event: EventError, Data: map[string]any{"message": msg}}
}

func doneEvent() SSEEvent {
	return SSEEvent{Event: EventDone, Data: map[string]any{}}
}
