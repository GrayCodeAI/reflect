package types

import (
	"context"

	"github.com/GrayCodeAI/mux/llm"
)

// ContentPart is a provider-neutral multimodal message part. Decoder owns this
// conversation shape; the Mux engine adapter translates it at the transport
// boundary. It aliases the canonical contract type.
type ContentPart = llm.ContentPart

// ImageURLPart describes an image URL or data URI.
type ImageURLPart = llm.ImageURLPart

// InputAudioPart describes base64-encoded audio content.
type InputAudioPart = llm.InputAudioPart

// ChatProvider is Decoder's transport-provider interface.
type ChatProvider interface {
	Chat(ctx context.Context, messages []MuxMessage, opts ChatOptions) (*MuxResponse, error)
	StreamChat(ctx context.Context, messages []MuxMessage, opts ChatOptions) (*StreamResult, error)
	Ping(ctx context.Context) error
	Name() string
}

// ResponseFormat specifies the desired output format for a Decoder runtime request.
type ResponseFormat = llm.ResponseFormat

// ToolChoiceOption controls how the model uses tools.
type ToolChoiceOption = llm.ToolChoiceOption

// MuxTool is Decoder's runtime tool definition DTO.
type MuxTool = llm.MuxTool

// ChatOptions holds Decoder-owned request options for an engine chat call.
type ChatOptions = llm.ChatOptions

// ContinuationConfig controls output continuation behavior for Decoder runtime calls.
type ContinuationConfig = llm.ContinuationConfig

// DefaultContinuationConfig is Decoder's agent-loop continuation policy. Mux
// receives these limits through its engine request rather than owning the
// product policy.
func DefaultContinuationConfig() ContinuationConfig {
	return ContinuationConfig{MaxContinuations: 3, MaxTotalTokens: 32000}
}

// ToolCall is Decoder's runtime tool invocation DTO.
type ToolCall = llm.ToolCall

// ToolResult is Decoder's runtime tool result DTO.
type ToolResult = llm.ToolResult

// MuxUsage tracks token usage for Decoder runtime responses and streams.
type MuxUsage = llm.MuxUsage

// ResolvedRoute is Decoder's view of the concrete provider/model route selected
// by the provider engine. Keeping this projection Decoder-owned prevents Mux's
// transport DTOs from leaking into product state and observability payloads.
type ResolvedRoute = llm.ResolvedRoute

// MuxResponse is Decoder's runtime chat response DTO.
type MuxResponse = llm.MuxResponse

// MuxStreamEvent is Decoder's runtime stream event DTO.
type MuxStreamEvent = llm.MuxStreamEvent

// StreamResult wraps a Decoder-owned streaming response with cleanup. It aliases
// the canonical contract type; its Close() method and canonical constructor
// (NewStreamResult) live in github.com/GrayCodeAI/mux/llm.
type StreamResult = llm.StreamResult

// MuxMessage is Decoder's runtime conversation DTO.
// It intentionally mirrors the engine boundary shape while remaining Decoder-owned.
type MuxMessage = llm.MuxMessage
