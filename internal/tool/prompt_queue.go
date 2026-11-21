// prompt_queue.go — Queue prompts for later processing.
//
// Allows users to queue multiple prompts that will be processed
// sequentially without manual intervention.

package tool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

// PromptQueueItem represents a queued prompt to be processed.
type PromptQueueItem struct {
	ID        string `json:"id,omitempty"`         // Unique identifier (timestamp-based)
	Prompt    string `json:"prompt,omitempty"`     // The prompt text
	Subject   string `json:"subject,omitempty"`    // Optional short subject line
	CreatedAt string `json:"created_at,omitempty"` // ISO timestamp
}

// PromptQueueState holds persisted queue data.
type PromptQueueState struct {
	Items []PromptQueueItem `json:"items,omitempty"`
}

var (
	queueState *PromptQueueState
	queueMu    sync.RWMutex
)

// queueStatePath returns the path to the prompt queue state file.
func queueStatePath() string {
	return filepath.Join(storage.StateDir(), "prompt_queue.json")
}

// loadPromptQueueLocked returns the in-memory queue state, loading it from
// disk on first use. Callers must already hold queueMu: every exported
// accessor in this file must take the mutex exactly once — the historical
// re-entrant locking here deadlocked EnqueuePrompt, DequeuePrompt,
// GetPromptQueue, and RemovePromptFromQueue on first use.
func loadPromptQueueLocked() *PromptQueueState {
	if queueState != nil {
		return queueState
	}

	queueState = &PromptQueueState{}
	data, err := os.ReadFile(queueStatePath())
	if err != nil {
		return queueState // Missing or unreadable: empty state.
	}

	if err := json.Unmarshal(data, queueState); err != nil {
		// Malformed: discard the partially decoded state and start empty.
		queueState = &PromptQueueState{}
	}
	return queueState
}

// ResetPromptQueueForTest discards the cached in-memory queue state so the
// next access re-reads the on-disk file. Test isolation helper.
func ResetPromptQueueForTest() {
	queueMu.Lock()
	queueState = nil
	queueMu.Unlock()
}

// LoadPromptQueue loads the prompt queue from disk.
func LoadPromptQueue() *PromptQueueState {
	queueMu.Lock()
	defer queueMu.Unlock()
	return loadPromptQueueLocked()
}

// SavePromptQueue writes the prompt queue to disk. Callers must hold queueMu
// (all in-package writers do).
func SavePromptQueue(state *PromptQueueState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(queueStatePath(), data, 0o600)
}

// EnqueuePrompt adds a prompt to the queue.
func EnqueuePrompt(prompt, subject string) error {
	queueMu.Lock()
	defer queueMu.Unlock()

	state := loadPromptQueueLocked()

	item := NewPromptQueueItem(prompt, subject)
	state.Items = append(state.Items, item)

	return SavePromptQueue(state)
}

// NewPromptQueueItem creates a new queue item with an ID.
func NewPromptQueueItem(prompt, subject string) PromptQueueItem {
	return PromptQueueItem{
		ID:        generateQueueItemID(),
		Prompt:    prompt,
		Subject:   subject,
		CreatedAt: formatQueueTime(),
	}
}

// generateQueueItemID creates a unique ID based on timestamp.
func generateQueueItemID() string {
	return time.Now().Format("20060102-150405.000")
}

// formatQueueTime returns current time in ISO format for queue IDs.
func formatQueueTime() string {
	return time.Now().Format(time.RFC3339)
}

// DequeuePrompt removes and returns the first item from the queue.
func DequeuePrompt() (*PromptQueueItem, error) {
	queueMu.Lock()
	defer queueMu.Unlock()

	state := loadPromptQueueLocked()

	if len(state.Items) == 0 {
		return nil, nil
	}

	item := state.Items[0]
	state.Items = state.Items[1:]

	_ = SavePromptQueue(state) // Ignore save error for dequeue
	return &item, nil
}

// ClearPromptQueue removes all items from the queue.
func ClearPromptQueue() error {
	queueMu.Lock()
	defer queueMu.Unlock()

	state := &PromptQueueState{Items: nil}
	// Keep the in-memory cache in sync with the cleared file, otherwise
	// GetPromptQueue keeps serving the stale pre-clear items.
	queueState = state

	return SavePromptQueue(state)
}

// GetPromptQueue returns a copy of the current queue items.
func GetPromptQueue() []PromptQueueItem {
	// The lazy disk load mutates the shared cache, so this takes the full
	// lock rather than RLock.
	queueMu.Lock()
	defer queueMu.Unlock()

	state := loadPromptQueueLocked()
	items := make([]PromptQueueItem, len(state.Items))
	copy(items, state.Items)
	return items
}

// RemovePromptFromQueue removes a specific item by index.
func RemovePromptFromQueue(index int) error {
	queueMu.Lock()
	defer queueMu.Unlock()

	state := loadPromptQueueLocked()

	if index < 0 || index >= len(state.Items) {
		return nil // Invalid index, nothing to remove
	}

	state.Items = append(state.Items[:index], state.Items[index+1:]...)

	return SavePromptQueue(state)
}

// Count returns the number of items in the queue.
func (s *PromptQueueState) Count() int {
	return len(s.Items)
}

// SortBy creates a sorted copy of the queue by creation time (oldest first).
func (s *PromptQueueState) SortByCreatedAt() []PromptQueueItem {
	items := make([]PromptQueueItem, len(s.Items))
	copy(items, s.Items)
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items
}
