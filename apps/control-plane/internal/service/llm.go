package service

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

// ChatCompleter abstracts the OpenAI chat completion call so we can swap in a
// mock for local testing without touching the agentic loop logic.
type ChatCompleter interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}
