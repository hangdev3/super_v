package core

import (
	"errors"
	"testing"
)

type fakeClipboard struct {
	text      string
	readError error
	writes    []string
}

func (f *fakeClipboard) ReadText() (string, error) {
	return f.text, f.readError
}

func (f *fakeClipboard) WriteText(text string) error {
	f.writes = append(f.writes, text)
	f.text = text
	return nil
}

func TestManagerSyncClipboardRecordsNewTextAndPropagatesReadErrors(t *testing.T) {
	// Arrange
	clipboard := &fakeClipboard{text: "do sistema"}
	manager := NewManager(NewHistory(5), clipboard)

	// Act
	added, err := manager.SyncClipboard()
	clipboard.readError = errors.New("clipboard indisponivel")
	_, readErr := manager.SyncClipboard()

	// Assert
	if err != nil || !added {
		t.Fatalf("expected clipboard text to be recorded, added=%v err=%v", added, err)
	}
	if readErr == nil || readErr.Error() != "clipboard indisponivel" {
		t.Fatalf("expected clipboard read error to be propagated, got %v", readErr)
	}
}

func TestManagerSelectWritesSelectedTextAndIgnoresThatWriteDuringNextSync(t *testing.T) {
	// Arrange
	clipboard := &fakeClipboard{}
	manager := NewManager(NewHistory(5), clipboard)
	manager.History().Add("mais antigo")
	manager.History().Add("mais novo")
	selected := manager.History().Items()[1]

	// Act
	err := manager.Select(selected.ID)
	added, syncErr := manager.SyncClipboard()

	// Assert
	if err != nil {
		t.Fatalf("expected selection to write clipboard, got %v", err)
	}
	if len(clipboard.writes) != 1 || clipboard.writes[0] != "mais antigo" {
		t.Fatalf("expected selected text to be written, got %#v", clipboard.writes)
	}
	if syncErr != nil || added {
		t.Fatalf("expected manager write not to create a new history entry, added=%v err=%v", added, syncErr)
	}
}

func TestManagerSelectReturnsAnErrorForUnknownItem(t *testing.T) {
	// Arrange
	manager := NewManager(NewHistory(5), &fakeClipboard{})

	// Act
	err := manager.Select("missing")

	// Assert
	if !errors.Is(err, ErrHistoryItemNotFound) {
		t.Fatalf("expected ErrHistoryItemNotFound, got %v", err)
	}
}
