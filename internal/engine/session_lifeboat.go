package engine

import (
	"errors"
	"strings"

	"github.com/GrayCodeAI/reflect/internal/session"
	"github.com/GrayCodeAI/reflect/internal/types"
)

var errSessionNoPersistedID = errors.New("session has no persisted id yet")

// SaveLifeboat captures the current working memory of a session before
// context compaction, writing it beside the transcript. It is safe to call on
// a nil session or when there is no persisted id yet; in those cases it is a
// no-op.
func (s *Session) SaveLifeboat() (string, error) {
	if s == nil || s.Persistence() == nil {
		return "", errors.New("session unavailable")
	}
	id := strings.TrimSpace(s.Persistence().PersistID())
	if id == "" {
		return "", errSessionNoPersistedID
	}

	raw := s.Persistence().RawMessages()
	msgs := make([]types.RelayMessage, 0, len(raw))
	msgs = append(msgs, raw...)
	state := session.BuildLifeboat(id, msgs)
	return session.SaveLifeboat(id, state)
}

// LoadLifeboat restores a previously saved lifeboat for s, if one exists.
func (s *Session) LoadLifeboat() (session.LifeboatState, error) {
	if s == nil || s.Persistence() == nil {
		return session.LifeboatState{}, errors.New("session unavailable")
	}
	id := strings.TrimSpace(s.Persistence().PersistID())
	if id == "" {
		return session.LifeboatState{}, errSessionNoPersistedID
	}
	return session.LoadLifeboat(id)
}
