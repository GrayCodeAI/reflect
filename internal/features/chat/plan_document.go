package chat

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/storage"
)

// PlanDocumentPath scopes one Markdown plan to a project and conversation.
// Keep host-managed state outside the user's source tree.
func PlanDocumentPath(projectRoot, sessionID string) string {
	key := sha256.Sum256([]byte(sessionID))
	return filepath.Join(storage.PlansDir(projectRoot), fmt.Sprintf("session-%x", key[:16]), "plan.md")
}

// SavePlanDocument atomically replaces this conversation's plan. Other sessions
// and structured plans keep their own files.
func SavePlanDocument(projectRoot, sessionID, content string) (string, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("a session and a non-empty plan are required")
	}
	path := PlanDocumentPath(projectRoot, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create plan directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".plan-*")
	if err != nil {
		return "", fmt.Errorf("create plan: %w", err)
	}
	defer func() { _ = file.Close(); _ = os.Remove(file.Name()) }()
	if _, err = file.WriteString(strings.TrimSpace(content) + "\n"); err != nil {
		return "", fmt.Errorf("write plan: %w", err)
	}
	if err = file.Close(); err != nil {
		return "", fmt.Errorf("close plan: %w", err)
	}
	if err = os.Rename(file.Name(), path); err != nil {
		return "", fmt.Errorf("save plan: %w", err)
	}
	return path, nil
}
