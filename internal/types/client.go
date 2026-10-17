package types

import (
	"context"

	"github.com/GrayCodeAI/relay/llm"
)

// ContentPart is a provider-neutral multimodal message part. Reflect owns this
// conversation shape; the Relay engine adapter translates it at the transport
// boundary. It aliases the canonical contract type.
type ContentPart = llm.ContentPart

// ImageURLPart describes an image URL or data URI.
type ImageURLPart = llm.ImageURLPart

// InputAudioPart describes base64-encoded audio content.
type InputAudioPart = llm.InputAudioPart

// ChatProvider is Reflect's transport-provider interface.
type ChatProvider interface {
	Chat(ctx context.Context, messages []RelayMessage, opts ChatOptions) (*RelayResponse, error)
	StreamChat(ctx context.Context, messages []RelayMessage, opts ChatOptions) (*StreamResult, error)
	Ping(ctx context.Context) error
	Name() string
}

// ResponseFormat specifies the desired output format for a Reflect runtime request.
type ResponseFormat = llm.ResponseFormat

// ToolChoiceOption controls how the model uses tools.
type ToolChoiceOption = llm.ToolChoiceOption

// RelayTool is Reflect's runtime tool definition DTO.
type RelayTool = llm.RelayTool

// ChatOptions holds Reflect-owned request options for an engine chat call.
type ChatOptions = llm.ChatOptions

// ContinuationConfig controls output continuation behavior for Reflect runtime calls.
type ContinuationConfig = llm.ContinuationConfig

// DefaultContinuationConfig is Reflect's agent-loop continuation policy. Relay
// receives these limits through its engine request rather than owning the
// product policy.
func DefaultContinuationConfig() ContinuationConfig {
	return ContinuationConfig{MaxContinuations: 3, MaxTotalTokens: 32000}
}

// ToolCall is Reflect's runtime tool invocation DTO.
type ToolCall = llm.ToolCall

// ToolResult is Reflect's runtime tool result DTO.
type ToolResult = llm.ToolResult

// RelayUsage tracks token usage for Reflect runtime responses and streams.
type RelayUsage = llm.RelayUsage

// ResolvedRoute is Reflect's view of the concrete provider/model route selected
// by the provider engine. Keeping this projection Reflect-owned prevents Relay's
// transport DTOs from leaking into product state and observability payloads.
type ResolvedRoute = llm.ResolvedRoute

// RelayResponse is Reflect's runtime chat response DTO.
type RelayResponse = llm.RelayResponse

// RelayStreamEvent is Reflect's runtime stream event DTO.
type RelayStreamEvent = llm.RelayStreamEvent

// StreamResult wraps a Reflect-owned streaming response with cleanup. It aliases
// the canonical contract type; its Close() method and canonical constructor
// (NewStreamResult) live in github.com/GrayCodeAI/relay/llm.
type StreamResult = llm.StreamResult

// RelayMessage is Reflect's runtime conversation DTO.
// It intentionally mirrors the engine boundary shape while remaining Reflect-owned.
type RelayMessage = llm.RelayMessage
