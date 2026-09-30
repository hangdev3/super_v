package core

import (
	"errors"
	"sync"
)

var ErrHistoryItemNotFound = errors.New("history item not found")
var ErrImageClipboardUnsupported = errors.New("image clipboard is not supported")

// Clipboard is the smallest platform boundary required by the core use cases.
type Clipboard interface {
	ReadText() (string, error)
	WriteText(text string) error
}

// ImageClipboard is the optional clipboard capability used by image history
// items. Text-only clipboard implementations remain valid.
type ImageClipboard interface {
	WriteImage(path string) error
}

// ImageClipboardReader is an optional platform capability. The boolean is
// false when the current clipboard selection is not an image, so text-only
// clipboard implementations remain valid.
type ImageClipboardReader interface {
	ReadImage() (path string, found bool, err error)
}

// Manager coordinates the history with the system clipboard.
type Manager struct {
	mu               sync.Mutex
	history          *History
	clipboard        Clipboard
	ignoreNext       string
	ignoreNextImage  bool
	ignoredImagePath string
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
	m.mu.Lock()
	defer m.mu.Unlock()

	if imageClipboard, ok := m.clipboard.(ImageClipboardReader); ok {
		path, found, err := imageClipboard.ReadImage()
		if err != nil {
			return false, err
		}
		if found {
			if m.ignoreNextImage {
				m.ignoreNextImage = false
				m.ignoredImagePath = path
				m.ignoreNext = ""
				return false, nil
			}
			if m.ignoredImagePath != "" {
				if path == m.ignoredImagePath {
					return false, nil
				}
				m.ignoredImagePath = ""
			}
			m.ignoreNext = ""
			return m.history.AddImage(path), nil
		}
	}

	text, err := m.clipboard.ReadText()
	if err != nil {
		return false, err
	}

	m.ignoreNextImage = false
	m.ignoredImagePath = ""
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
	if item.Kind == ClipKindImage {
		imageClipboard, ok := m.clipboard.(ImageClipboard)
		if !ok {
			return ErrImageClipboardUnsupported
		}
		if err := imageClipboard.WriteImage(item.ImagePath); err != nil {
			return err
		}
		m.ignoreNext = ""
		m.ignoreNextImage = true
		m.ignoredImagePath = ""
		return nil
	}

	if err := m.clipboard.WriteText(item.Text); err != nil {
		return err
	}

	m.ignoreNextImage = false
	m.ignoredImagePath = ""
	m.ignoreNext = item.Text
	return nil
}

// RecordCapturedImage stores a newly created screenshot and publishes that
// same file to the system clipboard. The next passive image read is ignored so
// the monitor does not add a second row for the clipboard echo.
func (m *Manager) RecordCapturedImage(path string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	added := m.history.AddImage(path)
	imageClipboard, ok := m.clipboard.(ImageClipboard)
	if !ok {
		return added, ErrImageClipboardUnsupported
	}
	if err := imageClipboard.WriteImage(path); err != nil {
		return added, err
	}

	m.ignoreNext = ""
	m.ignoreNextImage = true
	m.ignoredImagePath = ""
	return added, nil
}
