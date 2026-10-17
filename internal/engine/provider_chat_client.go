package engine

import (
	"context"
	"errors"

	"github.com/GrayCodeAI/reflect/internal/types"
)

// NewUnavailableChatClient preserves Session construction while surfacing
// Relay transport setup failures at the first chat call.
func NewUnavailableChatClient(err error) ChatClient {
	if err == nil {
		err = errors.New("reflect: chat transport unavailable")
	}
	return &unavailableChatClient{err: err}
}

type unavailableChatClient struct {
	err error
}

func (c *unavailableChatClient) Chat(context.Context, []types.RelayMessage, types.ChatOptions) (*types.RelayResponse, error) {
	return nil, c.err
}

func (c *unavailableChatClient) StreamChatContinue(context.Context, []types.RelayMessage, types.ChatOptions, types.ContinuationConfig) (*types.StreamResult, error) {
	return nil, c.err
}
