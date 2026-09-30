package core

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultMaxItems = 50

// ClipItem is one text entry captured from the system clipboard.
type ClipItem struct {
	ID        string
	Text      string
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
	h.mu.Lock()
	defer h.mu.Unlock()

	if strings.TrimSpace(text) == "" || h.containsText(text) {
		return false
	}

	h.nextID++
	item := ClipItem{
		ID:        fmt.Sprintf("clip-%d", h.nextID),
		Text:      text,
		CreatedAt: time.Now(),
	}
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
		if item.Text == text {
			return true
		}
	}

	return false
}
