package engine

import (
	"context"
	"sync"

	"github.com/GrayCodeAI/reflect/internal/types"
)

// mockClient implements ChatClient for testing without real LLM calls.
type mockClient struct {
	mu        sync.Mutex
	responses []*types.RelayResponse
	idx       int
	calls     []mockCall
}

type mockCall struct {
	method   string
	messages []types.RelayMessage
}

func newMockClient(responses ...*types.RelayResponse) *mockClient {
	return &mockClient{responses: responses}
}

func mockTextResponse(text string) *types.RelayResponse {
	return &types.RelayResponse{
		Content:      text,
		FinishReason: "end_turn",
		Usage:        &types.RelayUsage{PromptTokens: 50, CompletionTokens: 20, TotalTokens: 70},
	}
}

func (m *mockClient) Chat(ctx context.Context, messages []types.RelayMessage, opts types.ChatOptions) (*types.RelayResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, mockCall{method: "Chat", messages: messages})

	if m.idx >= len(m.responses) {
		return mockTextResponse("mock fallback response"), nil
	}
	resp := m.responses[m.idx]
	m.idx++
	return resp, nil
}

func (m *mockClient) StreamChatContinue(ctx context.Context, messages []types.RelayMessage, opts types.ChatOptions, cfg types.ContinuationConfig) (*types.StreamResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, mockCall{method: "StreamChatContinue", messages: messages})

	ch := make(chan types.RelayStreamEvent, 10)

	var content string
	var finishReason string
	if m.idx < len(m.responses) {
		resp := m.responses[m.idx]
		m.idx++
		content = resp.Content
		finishReason = resp.FinishReason
	} else {
		content = "mock streamed response"
		finishReason = "end_turn"
	}

	ch <- types.RelayStreamEvent{Type: "content", Content: content}
	ch <- types.RelayStreamEvent{
		Type:       "done",
		StopReason: finishReason,
		Usage:      &types.RelayUsage{PromptTokens: 50, CompletionTokens: 20, TotalTokens: 70},
	}
	close(ch)

	return &types.StreamResult{Events: ch}, nil
}

func (m *mockClient) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}
