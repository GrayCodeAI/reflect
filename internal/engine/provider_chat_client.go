package engine

import (
	"context"
	"errors"

	"github.com/GrayCodeAI/decoder/internal/types"
)

// NewUnavailableChatClient preserves Session construction while surfacing
// Mux transport setup failures at the first chat call.
func NewUnavailableChatClient(err error) ChatClient {
	if err == nil {
		err = errors.New("decoder: chat transport unavailable")
	}
	return &unavailableChatClient{err: err}
}

type unavailableChatClient struct {
	err error
}

func (c *unavailableChatClient) Chat(context.Context, []types.MuxMessage, types.ChatOptions) (*types.MuxResponse, error) {
	return nil, c.err
}

func (c *unavailableChatClient) StreamChatContinue(context.Context, []types.MuxMessage, types.ChatOptions, types.ContinuationConfig) (*types.StreamResult, error) {
	return nil, c.err
}
