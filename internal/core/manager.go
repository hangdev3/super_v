package core

import (
	"errors"
	"sync"
)

var ErrHistoryItemNotFound = errors.New("history item not found")

// Clipboard is the smallest platform boundary required by the core use cases.
type Clipboard interface {
	ReadText() (string, error)
	WriteText(text string) error
}

// Manager coordinates the history with the system clipboard.
type Manager struct {
	mu         sync.Mutex
	history    *History
	clipboard  Clipboard
	ignoreNext string
}

func NewManager(history *History, clipboard Clipboard) *Manager {
	if history == nil {
		history = NewHistory(defaultMaxItems)
	}

	return &Manager{
		history:   history,
		clipboard: clipboard,
	}
}

// History exposes the history for read-only UI operations and explicit clear.
func (m *Manager) History() *History {
	return m.history
}

// SyncClipboard records the current system clipboard when it is new.
func (m *Manager) SyncClipboard() (bool, error) {
	text, err := m.clipboard.ReadText()
	if err != nil {
		return false, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if text == m.ignoreNext {
		m.ignoreNext = ""
		return false, nil
	}

	m.ignoreNext = ""
	return m.history.Add(text), nil
}

// Select copies a history item back to the system clipboard.
func (m *Manager) Select(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, found := m.history.Select(id)
	if !found {
		return ErrHistoryItemNotFound
	}
	if err := m.clipboard.WriteText(item.Text); err != nil {
		return err
	}

	m.ignoreNext = item.Text
	return nil
}
