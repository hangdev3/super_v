package core

import (
	"errors"
	"testing"
)

type fakeClipboard struct {
	text      string
	readError error
	writes    []string
	images    []string
}

func (f *fakeClipboard) ReadText() (string, error) {
	return f.text, f.readError
}

func (f *fakeClipboard) WriteText(text string) error {
	f.writes = append(f.writes, text)
	f.text = text
	return nil
}

func (f *fakeClipboard) WriteImage(path string) error {
	f.images = append(f.images, path)
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

func TestManagerSyncClipboardNeverWritesBackToTheSystemClipboard(t *testing.T) {
	// Arrange
	clipboard := &fakeClipboard{text: "texto recebido"}
	manager := NewManager(NewHistory(5), clipboard)

	// Act
	_, err := manager.SyncClipboard()

	// Assert
	if err != nil {
		t.Fatalf("expected passive clipboard synchronization to succeed, got %v", err)
	}
	if len(clipboard.writes) != 0 || len(clipboard.images) != 0 {
		t.Fatalf("expected passive synchronization not to modify clipboard, writes=%#v images=%#v", clipboard.writes, clipboard.images)
	}
}

type fakeImageReaderClipboard struct {
	fakeClipboard
	imagePath  string
	imageFound bool
	imageError error
}

func (f *fakeImageReaderClipboard) ReadImage() (string, bool, error) {
	return f.imagePath, f.imageFound, f.imageError
}

func TestManagerSyncClipboardRecordsAnImageWithoutWritingItBack(t *testing.T) {
	// Arrange
	clipboard := &fakeImageReaderClipboard{
		imagePath:  "/home/look/Imagens/Captures/clipboard-image.png",
		imageFound: true,
	}
	manager := NewManager(NewHistory(5), clipboard)

	// Act
	added, err := manager.SyncClipboard()

	// Assert
	if err != nil || !added {
		t.Fatalf("expected clipboard image to be recorded, added=%v err=%v", added, err)
	}
	items := manager.History().Items()
	if len(items) != 1 || items[0].Kind != ClipKindImage || items[0].ImagePath != clipboard.imagePath {
		t.Fatalf("expected image history item, got %#v", items)
	}
	if len(clipboard.writes) != 0 || len(clipboard.images) != 0 {
		t.Fatalf("expected image synchronization not to modify clipboard, writes=%#v images=%#v", clipboard.writes, clipboard.images)
	}
}

func TestManagerIgnoresTheImageEchoAfterExplicitHistorySelection(t *testing.T) {
	// Arrange
	clipboard := &fakeImageReaderClipboard{}
	manager := NewManager(NewHistory(5), clipboard)
	manager.History().AddImage("/home/look/Imagens/Captures/original.png")
	item := manager.History().Items()[0]

	// Act
	if err := manager.Select(item.ID); err != nil {
		t.Fatalf("select image: %v", err)
	}
	clipboard.imagePath = "/home/look/Imagens/Captures/clipboard-copy.png"
	clipboard.imageFound = true
	added, err := manager.SyncClipboard()
	secondAdded, secondErr := manager.SyncClipboard()

	// Assert
	if err != nil {
		t.Fatalf("sync image echo: %v", err)
	}
	if secondErr != nil {
		t.Fatalf("sync repeated image echo: %v", secondErr)
	}
	if added || secondAdded {
		t.Fatalf("expected the clipboard image echo to remain ignored, added=%v secondAdded=%v", added, secondAdded)
	}
	if len(manager.History().Items()) != 1 {
		t.Fatalf("expected explicit image selection not to create a duplicate history row")
	}

	clipboard.imagePath = "/home/look/Imagens/Captures/new-image.png"
	added, err = manager.SyncClipboard()
	if err != nil || !added {
		t.Fatalf("expected a different clipboard image to be recorded, added=%v err=%v", added, err)
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

func TestManagerSelectWritesAnImageToTheClipboard(t *testing.T) {
	// Arrange
	clipboard := &fakeClipboard{}
	manager := NewManager(NewHistory(5), clipboard)
	manager.History().AddImage("/tmp/capture.png")
	item := manager.History().Items()[0]

	// Act
	err := manager.Select(item.ID)

	// Assert
	if err != nil {
		t.Fatalf("expected image selection to write clipboard, got %v", err)
	}
	if len(clipboard.images) != 1 || clipboard.images[0] != "/tmp/capture.png" {
		t.Fatalf("expected selected image to be written, got %#v", clipboard.images)
	}
}

func TestManagerRecordCapturedImageWritesAndRecordsTheLatestImage(t *testing.T) {
	// Arrange
	clipboard := &fakeClipboard{}
	manager := NewManager(NewHistory(5), clipboard)
	path := "/home/look/Imagens/Captures/capture-latest.png"

	// Act
	added, err := manager.RecordCapturedImage(path)

	// Assert
	if err != nil || !added {
		t.Fatalf("expected captured image to be recorded and copied, added=%v err=%v", added, err)
	}
	if len(clipboard.images) != 1 || clipboard.images[0] != path {
		t.Fatalf("expected latest capture to be written to clipboard, got %#v", clipboard.images)
	}
	items := manager.History().Items()
	if len(items) != 1 || items[0].Kind != ClipKindImage || items[0].ImagePath != path {
		t.Fatalf("expected latest capture in history, got %#v", items)
	}
}

func TestManagerRecordCapturedImageIgnoresItsClipboardEcho(t *testing.T) {
	// Arrange
	clipboard := &fakeImageReaderClipboard{}
	manager := NewManager(NewHistory(5), clipboard)
	path := "/home/look/Imagens/Captures/capture-latest.png"

	// Act
	if _, err := manager.RecordCapturedImage(path); err != nil {
		t.Fatalf("record captured image: %v", err)
	}
	clipboard.imagePath = "/home/look/Imagens/Captures/clipboard-latest.png"
	clipboard.imageFound = true
	added, err := manager.SyncClipboard()

	// Assert
	if err != nil {
		t.Fatalf("sync captured image echo: %v", err)
	}
	if added {
		t.Fatalf("expected captured image echo not to create a second history entry")
	}
	if len(manager.History().Items()) != 1 {
		t.Fatalf("expected one captured image in history, got %d", len(manager.History().Items()))
	}
}

func TestManagerSelectReturnsUnsupportedErrorWhenClipboardCannotWriteImages(t *testing.T) {
	// Arrange
	manager := NewManager(NewHistory(5), &textOnlyClipboard{})
	manager.History().AddImage("/tmp/capture.png")
	item := manager.History().Items()[0]

	// Act
	err := manager.Select(item.ID)

	// Assert
	if !errors.Is(err, ErrImageClipboardUnsupported) {
		t.Fatalf("expected unsupported image clipboard error, got %v", err)
	}
}

type textOnlyClipboard struct{}

func (textOnlyClipboard) ReadText() (string, error) { return "", nil }

func (textOnlyClipboard) WriteText(string) error { return nil }
