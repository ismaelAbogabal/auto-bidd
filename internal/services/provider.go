package services

import "context"

// Message represents a chat message for the LLM provider
type Message struct {
	Role            string
	Content         string
	CacheBreakpoint bool // Anthropic-only: mark as cache breakpoint
}

// LLMProvider abstracts the LLM API call layer
type LLMProvider interface {
	// Call sends a non-streaming request and returns the response text
	Call(ctx context.Context, systemPrompt string, messages []Message, maxTokens int) (string, error)

	// CallStream sends a streaming request, calling onText for each text chunk.
	// Returns the full accumulated text.
	CallStream(ctx context.Context, systemPrompt string, messages []Message, maxTokens int, onText func(string)) (string, error)
}
