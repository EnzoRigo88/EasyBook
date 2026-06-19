package service

import (
	"context"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// MockChatCompleter is a deterministic stand-in for the OpenAI API.
// It inspects the last user message for booking/cancellation keywords and
// returns scripted tool-call or text responses so the full HTTP → agent →
// booking-stub → response loop can run with zero network calls.
type MockChatCompleter struct {
	// callCount tracks how many times CreateChatCompletion has been called
	// within a single ProcessMessage invocation (reset per-request via closure).
	callCount int
	lastArgs  string
}

func NewMockChatCompleter() *MockChatCompleter {
	return &MockChatCompleter{}
}

func (m *MockChatCompleter) CreateChatCompletion(_ context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	m.callCount++

	userMsg := lastUserMessage(req.Messages)
	lower := strings.ToLower(userMsg)

	// On the first call decide the intent; on subsequent calls return text.
	if m.callCount == 1 {
		switch {
		case containsAny(lower, "reserv", "book", "cancha", "turno", "quiero"):
			// Step 1: check_availability
			m.lastArgs = `{"court_name":"Cancha 1","date":"2026-06-19","time":"19:00"}`
			return toolCallResponse("call_mock_avail_1", "check_availability", m.lastArgs), nil

		case containsAny(lower, "cancel"):
			m.lastArgs = `{"confirmed":true}`
			return toolCallResponse("call_mock_cancel_1", "cancel_booking", m.lastArgs), nil

		case containsAny(lower, "disponib", "horario", "libre"):
			m.lastArgs = `{"court_name":"Cancha 1","date":"2026-06-19","time":"19:00"}`
			return toolCallResponse("call_mock_avail_2", "check_availability", m.lastArgs), nil
		}
	}

	if m.callCount == 2 && containsAny(strings.ToLower(userMsg), "reserv", "book", "cancha", "turno", "quiero") {
		// Step 2: create_booking after availability confirmed
		args := `{"court_id":"court-mock-1","date":"2026-06-19","time":"19:00","confirmed":true}`
		return toolCallResponse("call_mock_book_1", "create_booking", args), nil
	}

	// Final text response
	return textResponse("¡Listo! Tu reserva para mañana a las 19:00 en Cancha 1 está confirmada. ¡Hasta entonces!"), nil
}

// lastUserMessage returns the content of the last user-role message in the list.
func lastUserMessage(msgs []openai.ChatCompletionMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == openai.ChatMessageRoleUser {
			return msgs[i].Content
		}
	}
	return ""
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func toolCallResponse(id, name, args string) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleAssistant,
					ToolCalls: []openai.ToolCall{
						{
							ID:   id,
							Type: openai.ToolTypeFunction,
							Function: openai.FunctionCall{
								Name:      name,
								Arguments: args,
							},
						},
					},
				},
			},
		},
	}
}

func textResponse(content string) openai.ChatCompletionResponse {
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: content,
				},
			},
		},
	}
}
