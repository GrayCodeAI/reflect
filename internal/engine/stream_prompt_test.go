package engine

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GrayCodeAI/reflect/internal/bridge/sessioncapture"
	"github.com/GrayCodeAI/reflect/internal/engine/safety"
	"github.com/GrayCodeAI/reflect/internal/tool"
	"github.com/GrayCodeAI/reflect/internal/types"
)

func TestIsSmallTalkPrompt(t *testing.T) {
	tests := []struct {
		prompt string
		skip   bool
	}{
		{prompt: "Hi", skip: true},
		{prompt: "Hello!", skip: true},
		{prompt: "what can you do?", skip: true},
		{prompt: "how's it going", skip: true},
		{prompt: "good morning", skip: true},
		{prompt: "thanks", skip: true},
		{prompt: "Hi, inspect this repository", skip: false},
		{prompt: "Hi fix bug", skip: false},
		{prompt: "hi run ls", skip: false},
		{prompt: "run the tests", skip: false},
		{prompt: "hello there, who fixes this bug?", skip: false},
	}
	for _, tt := range tests {
		t.Run(tt.prompt, func(t *testing.T) {
			if got := isSmallTalkPrompt(tt.prompt); got != tt.skip {
				t.Fatalf("isSmallTalkPrompt(%q) = %v, want %v", tt.prompt, got, tt.skip)
			}
		})
	}
}

type conversationalStreamClient struct {
	*resilienceBoundaryClient
	options []types.ChatOptions
}

func (c *conversationalStreamClient) StreamChatContinue(ctx context.Context, messages []types.RelayMessage, opts types.ChatOptions, cfg types.ContinuationConfig) (*types.StreamResult, error) {
	c.options = append(c.options, opts)
	return c.resilienceBoundaryClient.StreamChatContinue(ctx, messages, opts, cfg)
}

func TestConversationStaysToolFreeAfterTerminalContextAndHistory(t *testing.T) {
	for _, tc := range []struct {
		name    string
		history bool
		content string
	}{
		{name: "fresh", content: "Hi! What would you like to work on?"},
		{name: "after tool use", history: true, content: "Hi! What would you like to work on?"},
		{name: "provider returned only tools"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			read := &countedReadTool{}
			client := &conversationalStreamClient{resilienceBoundaryClient: &resilienceBoundaryClient{streams: [][]types.RelayStreamEvent{{
				{Type: "content", Content: tc.content},
				// A provider may ignore tool_choice=none. The host must still
				// avoid repository inspection and tool activity in the UI.
				{Type: "tool_call", ToolCall: &types.ToolCall{ID: "unexpected", Name: "Read", Arguments: map[string]interface{}{"id": 1}}},
				{Type: "done", StopReason: "tool_use"},
			}}}}
			s := NewSessionWithClient(client, "test", "test-model", "WORKSPACE_INSTRUCTIONS", tool.NewRegistry(read), false)
			s.PermSvc().SetAutonomy(safety.AutonomyYOLO)
			if tc.history {
				s.AddUser("read a file")
				s.Persistence().AppendAssistantJournaled(types.RelayMessage{Role: "assistant", ToolUse: []types.ToolCall{{ID: "prior", Name: "Read"}}})
				s.Persistence().AppendUserJournaled(types.RelayMessage{Role: "user", ToolResults: []types.ToolResult{{ToolUseID: "prior", Content: "file contents"}}})
				s.AddAssistant("Read the file.")
			}
			s.AddUser(sessioncapture.NewTerminalContext().BuildContext("Hi"))
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			events, err := s.Stream(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var reply strings.Builder
			for event := range events {
				switch event.Type {
				case "tool_use", "tool_result", "error":
					t.Errorf("greeting emitted %s: %s", event.Type, event.Content)
				case "content":
					reply.WriteString(event.Content)
				}
			}
			if got := atomic.LoadInt32(&read.max); got != 0 {
				t.Fatalf("greeting executed a tool: %d", got)
			}
			wantReply := tc.content
			if wantReply == "" {
				wantReply = "I'm Reflect, developed by GrayCodeAI. How can I help?"
			}
			if reply.String() != wantReply || len(client.options) != 1 {
				t.Fatalf("unexpected greeting: %q, requests=%d", reply.String(), len(client.options))
			}
			opts := client.options[0]
			if len(opts.Tools) != 0 || opts.ToolChoice == nil || opts.ToolChoice.Type != "none" || strings.Contains(opts.System, "WORKSPACE_INSTRUCTIONS") {
				t.Fatalf("greeting exposed project tools or instructions: %+v", opts)
			}
		})
	}
}

func TestWorkFollowingGreetingKeepsTools(t *testing.T) {
	s := newMockSession(newMockClient())
	s.Tools().registry = tool.NewRegistry(&countedReadTool{})
	for _, text := range []string{"Hi fix bug", "hi run ls", "read main.go"} {
		opts := s.buildTurnOptions(turnContext{ctx: t.Context(), activeModel: "test-model", maxTok: 256, lastUserMsg: text, smallTalk: isSmallTalkPrompt(text)})
		if len(opts.Tools) == 0 || opts.ToolChoice != nil && opts.ToolChoice.Type == "none" {
			t.Fatalf("work lost tools: %q", text)
		}
	}
}
