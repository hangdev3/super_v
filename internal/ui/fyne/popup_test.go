package fyneui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
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

func (emptyPopupClipboard) WriteImage(string) error { return nil }

type syncingPopupClipboard struct {
	text string
}

func (c syncingPopupClipboard) ReadText() (string, error) { return c.text, nil }

func (syncingPopupClipboard) WriteText(string) error { return nil }

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

type recordingWindowController struct {
	activated chan uint32
}

func (c *recordingWindowController) HideDecorations(string) error { return nil }

func (c *recordingWindowController) IsFocused(string) (bool, error) { return true, nil }

func (c *recordingWindowController) Activate(_ string, timestamp uint32) error {
	c.activated <- timestamp
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

func TestPopupSynchronizesTheLatestClipboardContentBeforeRendering(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), syncingPopupClipboard{text: "copiado agora"})
	popup := NewPopup(application, manager, emptyPopupPointer{})

	// Act
	if err := popup.ShowAtPointer(); err != nil {
		t.Fatalf("show popup: %v", err)
	}
	popup.Hide()

	// Assert
	items := manager.History().Items()
	if len(items) != 1 || items[0].Text != "copiado agora" {
		t.Fatalf("expected latest clipboard content to be visible on first open, got %#v", items)
	}
}

func TestPopupUsesTimestampedWindowActivationForShortcutOpen(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})
	controller := &recordingWindowController{activated: make(chan uint32, 1)}
	popup.SetWindowController(controller)

	// Act
	popup.requestWindowActivation(123456)

	// Assert
	select {
	case timestamp := <-controller.activated:
		if timestamp != 123456 {
			t.Fatalf("activation timestamp = %d, want 123456", timestamp)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected timestamped window activation")
	}
}

func TestPopupUsesCompactLayoutAndMinimalDeleteAction(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})

	// Act
	item := newHistoryRow()

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
	if item.label.Truncation != fyne.TextTruncateClip {
		t.Fatalf("expected history rows to truncate long previews")
	}
}

func TestPopupUsesRoundedBackgroundWithNinetyPercentOpacity(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})

	// Act
	r, g, b, a := popup.background.FillColor.RGBA()
	wantR, wantG, wantB, wantA := popupBackgroundColor.RGBA()

	// Assert
	if popup.background.CornerRadius != popupCornerRadius {
		t.Fatalf("popup corner radius = %v, want %v", popup.background.CornerRadius, popupCornerRadius)
	}
	if r != wantR || g != wantG || b != wantB {
		t.Fatalf("popup background color = %#v, want %#v", color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}, popupBackgroundColor)
	}
	if a != wantA || uint8(a>>8) != popupBackgroundAlpha {
		t.Fatalf("popup background alpha = %d, want %d", a>>8, popupBackgroundAlpha)
	}
}

func TestPopupUpdatesHistoryItemText(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})
	popup.items = []core.ClipItem{{Text: "texto copiado"}}
	item := newHistoryRow()

	// Act
	popup.list.UpdateItem(0, item.Container)

	// Assert
	if item.label.Text != "texto copiado" {
		t.Fatalf("history item text = %q, want %q", item.label.Text, "texto copiado")
	}
}

func TestHistoryItemUsesNativeFyneContainer(t *testing.T) {
	if _, ok := newHistoryItem().(*fyne.Container); !ok {
		t.Fatalf("expected history items to be native Fyne containers")
	}
}

func TestHistoryRowReservesSpaceForTextAndImageContent(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	row := newHistoryRow()
	path := writeTestPNG(t)

	// Act
	row.Resize(fyne.NewSize(300, historyRowHeight))
	row.update(core.ClipItem{Kind: core.ClipKindImage, ImagePath: path})

	// Assert
	if row.Size().Height != historyRowHeight {
		t.Fatalf("history row height = %v, want %v", row.Size().Height, historyRowHeight)
	}
	if row.thumbnail.Size().Height < historyThumbnailSize.Height {
		t.Fatalf("thumbnail height = %v, want at least %v", row.thumbnail.Size().Height, historyThumbnailSize.Height)
	}
	if row.label.Size().Width <= 0 {
		t.Fatalf("expected the image filename label to receive horizontal space")
	}
}

func TestPopupRendersImageThumbnailAndFilename(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	path := writeTestPNG(t)
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	popup := NewPopup(application, manager, emptyPopupPointer{})
	popup.items = []core.ClipItem{{Kind: core.ClipKindImage, ImagePath: path}}
	item := newHistoryRow()

	// Act
	popup.list.UpdateItem(0, item.Container)

	// Assert
	if !item.thumbnail.Visible() {
		t.Fatalf("expected image thumbnail to be visible")
	}
	if item.thumbnail.File != path {
		t.Fatalf("thumbnail file = %q, want %q", item.thumbnail.File, path)
	}
	if item.label.Text != filepath.Base(path) {
		t.Fatalf("image row label = %q, want %q", item.label.Text, filepath.Base(path))
	}
}

func TestPopupCanvasActuallyRendersHistoryContent(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	path := writeTestPNG(t)
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	manager.History().AddImage(path)
	popup := NewPopup(application, manager, emptyPopupPointer{})

	// Act
	if err := popup.ShowAtPointer(); err != nil {
		t.Fatalf("show popup: %v", err)
	}
	rendered := popup.window.Canvas().Capture()
	popup.Hide()

	// Assert
	bluePixels := 0
	textPixels := 0
	for y := 0; y < rendered.Bounds().Dy(); y++ {
		for x := 0; x < rendered.Bounds().Dx(); x++ {
			r, g, b, _ := rendered.At(x, y).RGBA()
			if r>>8 == 40 && g>>8 == 80 && b>>8 == 120 {
				bluePixels++
			}
			if x >= 52 && y >= 40 && y < 120 && r>>8 > 160 && g>>8 > 160 && b>>8 > 160 {
				textPixels++
			}
		}
	}
	if bluePixels == 0 {
		t.Fatalf("expected the popup canvas to render the image thumbnail")
	}
	if textPixels == 0 {
		t.Fatalf("expected the popup canvas to render the image filename")
	}
}

func TestPopupDoesNotArmAutomaticPasteForImageSelection(t *testing.T) {
	// Arrange
	application := test.NewApp()
	defer application.Quit()
	path := writeTestPNG(t)
	manager := core.NewManager(core.NewHistory(5), emptyPopupClipboard{})
	manager.History().AddImage(path)
	popup := NewPopup(application, manager, emptyPopupPointer{})
	popup.items = manager.History().Items()

	// Act
	popup.selectItem(0)

	// Assert
	if popup.autoPasteArmed() {
		t.Fatalf("expected image selection not to arm automatic paste")
	}
	if popup.status.Text != "Imagem copiada — cole onde preferir" {
		t.Fatalf("image selection status = %q", popup.status.Text)
	}
}

func writeTestPNG(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "capture.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test image: %v", err)
	}
	defer file.Close()
	testImage := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			testImage.Set(x, y, color.RGBA{R: 40, G: 80, B: 120, A: 255})
		}
	}
	if err := png.Encode(file, testImage); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	return path
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

func TestPopupFocusLossWaitsForActivationGracePeriod(t *testing.T) {
	// Arrange
	lostAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)

	// Act
	tooSoon := popupFocusLossReady(lostAt, lostAt.Add(popupFocusLossGrace-time.Nanosecond))
	ready := popupFocusLossReady(lostAt, lostAt.Add(popupFocusLossGrace))

	// Assert
	if tooSoon {
		t.Fatalf("expected transient focus loss to be ignored during the grace period")
	}
	if !ready {
		t.Fatalf("expected stable focus loss to be accepted after the grace period")
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
