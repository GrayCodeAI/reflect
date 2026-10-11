package engine

import (
	"context"
	"sync"

	"github.com/GrayCodeAI/decoder/internal/types"
)

// mockClient implements ChatClient for testing without real LLM calls.
type mockClient struct {
	mu        sync.Mutex
	responses []*types.MuxResponse
	idx       int
	calls     []mockCall
}

type mockCall struct {
	method   string
	messages []types.MuxMessage
}

func newMockClient(responses ...*types.MuxResponse) *mockClient {
	return &mockClient{responses: responses}
}

func mockTextResponse(text string) *types.MuxResponse {
	return &types.MuxResponse{
		Content:      text,
		FinishReason: "end_turn",
		Usage:        &types.MuxUsage{PromptTokens: 50, CompletionTokens: 20, TotalTokens: 70},
	}
}

func (m *mockClient) Chat(ctx context.Context, messages []types.MuxMessage, opts types.ChatOptions) (*types.MuxResponse, error) {
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

func (m *mockClient) StreamChatContinue(ctx context.Context, messages []types.MuxMessage, opts types.ChatOptions, cfg types.ContinuationConfig) (*types.StreamResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, mockCall{method: "StreamChatContinue", messages: messages})

	ch := make(chan types.MuxStreamEvent, 10)

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

	ch <- types.MuxStreamEvent{Type: "content", Content: content}
	ch <- types.MuxStreamEvent{
		Type:       "done",
		StopReason: finishReason,
		Usage:      &types.MuxUsage{PromptTokens: 50, CompletionTokens: 20, TotalTokens: 70},
	}
	close(ch)

	return &types.StreamResult{Events: ch}, nil
}

func (m *mockClient) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}
