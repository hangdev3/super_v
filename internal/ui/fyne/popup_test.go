package fyneui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"super_v/internal/core"
	"super_v/internal/platform/pointer"
)

type emptyPopupClipboard struct{}

func (emptyPopupClipboard) ReadText() (string, error) { return "", nil }

func (emptyPopupClipboard) WriteText(string) error { return nil }

type emptyPopupPointer struct{}

func (emptyPopupPointer) Position() (pointer.Position, error) {
	return pointer.Position{X: 100, Y: 100}, nil
}

type recordingPaster struct {
	calls int
}

func (p *recordingPaster) Paste() error {
	p.calls++
	return nil
}

func TestPopupShowsAnExplicitEmptyStateWhenHistoryIsEmpty(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})

	// Act
	err := popup.ShowAtPointer()

	// Assert
	if err != nil {
		t.Fatalf("expected empty popup to open, got %v", err)
	}
	if !popup.emptyState.Visible() {
		t.Fatalf("expected the empty-state message to be visible")
	}
	if popup.list.Visible() {
		t.Fatalf("expected the clipboard list to be hidden when history is empty")
	}

	popup.Hide()
}

func TestPopupUsesCompactLayoutAndMinimalDeleteAction(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})

	// Act
	item := newHistoryItem().(*widget.Label)

	// Assert
	if popupSize != fyne.NewSize(342, 300) {
		t.Fatalf("expected compact popup size, got %v", popupSize)
	}
	if popup.clearButton.Text != "" {
		t.Fatalf("expected clear action to be icon-only, got text %q", popup.clearButton.Text)
	}
	if popup.clearButton.Icon == nil || popup.clearButton.Icon.Name() != theme.DeleteIcon().Name() {
		t.Fatalf("expected the clear action to use the delete icon")
	}
	if popup.clearButton.Importance != widget.LowImportance {
		t.Fatalf("expected the clear action to use low visual importance")
	}
	if popup.list.HideSeparators {
		t.Fatalf("expected history rows to show separators")
	}
	if item.Truncation != fyne.TextTruncateClip {
		t.Fatalf("expected history rows to truncate long previews")
	}
}

func TestPopupUpdatesHistoryItemText(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})
	popup.items = []core.ClipItem{{Text: "texto copiado"}}
	item := newHistoryItem()

	// Act
	popup.list.UpdateItem(0, item)

	// Assert
	label := item.(*widget.Label)
	if label.Text != "texto copiado" {
		t.Fatalf("history item text = %q, want %q", label.Text, "texto copiado")
	}
}

func TestHistoryStatusUsesCorrectSingularAndPlural(t *testing.T) {
	// Arrange
	wantSingular := "1 item — Enter ou clique para copiar"
	wantPlural := "2 itens — Enter ou clique para copiar"

	// Act
	singular := historyStatus(1)
	plural := historyStatus(2)

	// Assert
	if singular != wantSingular {
		t.Fatalf("singular history status = %q, want %q", singular, wantSingular)
	}
	if plural != wantPlural {
		t.Fatalf("plural history status = %q, want %q", plural, wantPlural)
	}
}

func TestPopupDismissalIgnoresPointerMovementAndHandlesFocusLoss(t *testing.T) {
	// Arrange
	// Act
	movedOutside := popupShouldDismiss(true, true)
	clickedOutside := popupShouldDismiss(false, true)
	focusUnknown := popupShouldDismiss(false, false)

	// Assert
	if movedOutside {
		t.Fatalf("expected pointer movement outside the popup not to dismiss it")
	}
	if !clickedOutside {
		t.Fatalf("expected a popup to dismiss when focus moves to another window")
	}
	if focusUnknown {
		t.Fatalf("expected an unknown focus state not to dismiss the popup")
	}
}

func TestPopupDismissalWaitsForMinimumVisibleDuration(t *testing.T) {
	// Arrange
	shownAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)

	// Act
	tooSoon := popupDismissalReady(shownAt, shownAt.Add(popupMinimumVisibleDuration-time.Nanosecond))
	ready := popupDismissalReady(shownAt, shownAt.Add(popupMinimumVisibleDuration))

	// Assert
	if tooSoon {
		t.Fatalf("expected dismissal to remain blocked during the minimum visible duration")
	}
	if !ready {
		t.Fatalf("expected dismissal to be allowed after the minimum visible duration")
	}
}

func TestPopupSelectionArmsAutomaticPasteWithoutHidingImmediately(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	manager.History().Add("texto para colar")
	popup := NewPopup(application, manager, emptyPopupPointer{})
	popup.items = manager.History().Items()
	paster := &recordingPaster{}
	popup.SetPaster(paster)

	// Act
	popup.selectItem(0)

	// Assert
	if !popup.autoPasteArmed() {
		t.Fatalf("expected selecting an item to arm automatic paste")
	}
	if !popup.autoPasteArmed() {
		t.Fatalf("expected the paste arm state to remain available until focus leaves")
	}
	if paster.calls != 0 {
		t.Fatalf("expected paste to wait for focus to leave, got %d calls", paster.calls)
	}
}
