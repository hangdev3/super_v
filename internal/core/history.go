package core

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultMaxItems = 10

type ClipKind string

const (
	ClipKindText  ClipKind = "text"
	ClipKindImage ClipKind = "image"
)

// ClipItem is one text or image entry captured from the system clipboard.
type ClipItem struct {
	ID        string
	Kind      ClipKind
	Text      string
	ImagePath string
	CreatedAt time.Time
}

// History stores the most recent clipboard entries in display order.
type History struct {
	mu       sync.RWMutex
	items    []ClipItem
	maxItems int
	nextID   uint64
}

// NewHistory creates an empty history. Non-positive limits use the product default.
func NewHistory(maxItems int) *History {
	if maxItems <= 0 {
		maxItems = defaultMaxItems
	}

	return &History{
		maxItems: maxItems,
		items:    make([]ClipItem, 0, maxItems),
	}
}

// Add records text at the beginning of the history unless it is blank or
// already present. It returns whether a new item was added.
func (h *History) Add(text string) bool {
	return h.AddText(text)
}

// AddText records text at the beginning of the history unless it is blank or
// already present.
func (h *History) AddText(text string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if strings.TrimSpace(text) == "" || h.containsText(text) {
		return false
	}

	h.nextID++
	item := ClipItem{
		ID:        fmt.Sprintf("clip-%d", h.nextID),
		Kind:      ClipKindText,
		Text:      text,
		CreatedAt: time.Now(),
	}
	return h.addItem(item)
}

// AddImage records an image file at the beginning of the history unless the
// path is blank or already present.
func (h *History) AddImage(path string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if strings.TrimSpace(path) == "" || h.containsImage(path) {
		return false
	}

	h.nextID++
	item := ClipItem{
		ID:        fmt.Sprintf("clip-%d", h.nextID),
		Kind:      ClipKindImage,
		ImagePath: path,
		CreatedAt: time.Now(),
	}
	return h.addItem(item)
}

func (h *History) addItem(item ClipItem) bool {
	h.items = append([]ClipItem{item}, h.items...)
	if len(h.items) > h.maxItems {
		h.items = h.items[:h.maxItems]
	}

	return true
}

// Items returns a copy in newest-first order.
func (h *History) Items() []ClipItem {
	h.mu.RLock()
	defer h.mu.RUnlock()

	items := make([]ClipItem, len(h.items))
	copy(items, h.items)
	return items
}

// Select finds an item by its stable history ID.
func (h *History) Select(id string) (ClipItem, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, item := range h.items {
		if item.ID == id {
			return item, true
		}
	}

	return ClipItem{}, false
}

// Clear removes every item while keeping the configured limit.
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.items = h.items[:0]
}

func (h *History) containsText(text string) bool {
	for _, item := range h.items {
		if item.Kind != ClipKindImage && item.Text == text {
			return true
		}
	}

	return false
}

func (h *History) containsImage(path string) bool {
	for _, item := range h.items {
		if item.Kind == ClipKindImage && item.ImagePath == path {
			return true
		}
	}

	return false
}
