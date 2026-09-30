package fyneui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"super_v/internal/core"
	"super_v/internal/platform/pointer"
)

type PointerSource interface {
	Position() (pointer.Position, error)
}

type Paster interface {
	Paste() error
}

type WindowController interface {
	HideDecorations(title string) error
	IsFocused(title string) (bool, error)
	Activate(title string, timestamp uint32) error
}

type Popup struct {
	manager     *core.Manager
	pointer     PointerSource
	window      fyne.Window
	background  *canvas.Rectangle
	body        *fyne.Container
	list        *widget.List
	clearButton *widget.Button
	emptyState  *widget.Label
	status      *widget.Label
	items       []core.ClipItem
	controller  WindowController
	paster      Paster
	dismissMu   sync.Mutex
	dismissStop chan struct{}
	pasteMu     sync.Mutex
	pasteArmed  bool
}

var popupSize = fyne.NewSize(342, 300)
var historyThumbnailSize = fyne.NewSize(48, 48)

var popupBackgroundColor = color.NRGBA{R: 24, G: 24, B: 27, A: popupBackgroundAlpha}

const (
	historyPreviewMaxRunes              = 48
	historyRowHeight                    = 64
	popupCornerRadius           float32 = 10
	popupBackgroundAlpha        uint8   = 230 // approximately 90%
	popupAnchorOffset                   = 12
	popupMinimumVisibleDuration         = 800 * time.Millisecond
	popupFocusLossGrace                 = 250 * time.Millisecond
	popupDismissPollPeriod              = 80 * time.Millisecond
)

func newHistoryItem() fyne.CanvasObject {
	return newHistoryRow().Container
}

func newHistoryRow() *historyRow {
	label := widget.NewLabel("")
	label.Truncation = fyne.TextTruncateClip
	thumbnail := canvas.NewImageFromImage(nil)
	thumbnail.FillMode = canvas.ImageFillContain
	thumbnail.ScaleMode = canvas.ImageScaleSmooth
	thumbnail.SetMinSize(historyThumbnailSize)
	thumbnail.Hide()
	return &historyRow{
		Container: container.NewBorder(nil, nil, thumbnail, nil, label),
		label:     label,
		thumbnail: thumbnail,
	}
}

type historyRow struct {
	*fyne.Container
	label     *widget.Label
	thumbnail *canvas.Image
}

func (r *historyRow) update(item core.ClipItem) {
	if item.Kind == core.ClipKindImage {
		r.label.SetText(filepath.Base(item.ImagePath))
		r.thumbnail.File = item.ImagePath
		r.thumbnail.Image = nil
		r.thumbnail.Resource = nil
		r.thumbnail.Show()
	} else {
		r.label.SetText(previewText(item.Text, historyPreviewMaxRunes))
		r.thumbnail.File = ""
		r.thumbnail.Image = nil
		r.thumbnail.Resource = nil
		r.thumbnail.Hide()
	}
	// The border layout is skipped while a child is hidden. Re-run it before
	// refreshing so the image receives a real 48x48 area and text gets the
	// complete row width when the thumbnail is removed.
	r.Layout.Layout(r.Objects, r.Size())
	if r.thumbnail.Visible() {
		r.thumbnail.Refresh()
	}
	r.Refresh()
}

func historyRowFromObject(object fyne.CanvasObject) (*historyRow, bool) {
	row, ok := object.(*fyne.Container)
	if !ok {
		return nil, false
	}

	var label *widget.Label
	var thumbnail *canvas.Image
	for _, child := range row.Objects {
		switch typedChild := child.(type) {
		case *widget.Label:
			label = typedChild
		case *canvas.Image:
			thumbnail = typedChild
		}
	}
	if label == nil || thumbnail == nil {
		return nil, false
	}

	return &historyRow{Container: row, label: label, thumbnail: thumbnail}, true
}

func NewPopup(application fyne.App, manager *core.Manager, pointerSource PointerSource) *Popup {
	p := &Popup{
		manager:    manager,
		pointer:    pointerSource,
		emptyState: widget.NewLabel("Nenhum item copiado ainda"),
		status:     widget.NewLabel("Histórico vazio"),
	}

	p.list = widget.NewList(
		func() int { return len(p.items) },
		newHistoryItem,
		func(id widget.ListItemID, object fyne.CanvasObject) {
			row, ok := historyRowFromObject(object)
			if !ok || id < 0 || id >= len(p.items) {
				return
			}
			row.update(p.items[id])
		},
	)
	p.list.HideSeparators = false
	p.list.OnSelected = func(id widget.ListItemID) {
		p.selectItem(id)
	}

	p.clearButton = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		p.manager.History().Clear()
		p.refreshItems()
	})
	p.clearButton.Importance = widget.LowImportance
	headerTitle := widget.NewLabelWithStyle("Recentes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	header := container.NewBorder(nil, nil, nil, p.clearButton, headerTitle)
	p.window = application.NewWindow("Super+Shift+V — histórico")
	p.body = container.NewStack(p.list, container.NewCenter(p.emptyState))
	p.background = canvas.NewRectangle(popupBackgroundColor)
	p.background.CornerRadius = popupCornerRadius
	content := container.NewBorder(header, p.status, nil, nil, p.body)
	p.window.SetContent(container.NewStack(p.background, container.NewPadded(content)))
	p.window.SetPadded(false)
	p.window.SetCloseIntercept(p.Hide)
	p.window.SetOnClosed(func() { p.stopDismissalMonitor() })
	p.window.Resize(popupSize)
	p.refreshItems()

	return p
}

func (p *Popup) SetWindowController(controller WindowController) {
	p.controller = controller
}

func (p *Popup) SetPaster(paster Paster) {
	p.paster = paster
}

func (p *Popup) ShowAtPointer(activationTimestamps ...uint32) error {
	position, err := p.pointer.Position()
	if err != nil {
		return err
	}

	// The background monitor is intentionally polled, but the user expects the
	// content copied immediately before the shortcut to be present on the first
	// popup open. A synchronous best-effort read closes that polling window for
	// both text and image clipboard content without writing anything back.
	_, _ = p.manager.SyncClipboard()
	p.refreshItems()
	p.disarmAutoPaste()
	if desktopWindow, ok := p.window.(desktop.Window); ok {
		desktopWindow.RequestAlwaysOnTop()
		desktopWindow.RequestPosition(position.X+popupAnchorOffset, position.Y+popupAnchorOffset)
	}
	p.window.Show()
	p.requestWindowActivation(firstTimestamp(activationTimestamps))
	p.startDismissalMonitor()
	return nil
}

func firstTimestamp(timestamps []uint32) uint32 {
	if len(timestamps) == 0 {
		return 0
	}
	return timestamps[0]
}

func (p *Popup) requestWindowActivation(timestamp uint32) {
	if p.controller == nil || timestamp == 0 {
		p.window.RequestFocus()
		p.hideWindowDecorations()
		return
	}

	title := p.window.Title()
	go func() {
		for attempt := 0; attempt < 12; attempt++ {
			_ = p.controller.HideDecorations(title)
			if err := p.controller.Activate(title, timestamp); err == nil {
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
	}()
}

func (p *Popup) Hide() {
	p.stopDismissalMonitor()
	p.window.Hide()
}

func (p *Popup) startDismissalMonitor() {
	p.stopDismissalMonitor()

	stop := make(chan struct{})
	p.dismissMu.Lock()
	p.dismissStop = stop
	p.dismissMu.Unlock()

	shownAt := time.Now()
	var focusLostAt time.Time
	go func() {
		ticker := time.NewTicker(popupDismissPollPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case now := <-ticker.C:
				if !popupDismissalReady(shownAt, now) {
					continue
				}

				focused, focusKnown := true, false
				if p.controller != nil {
					var err error
					focused, err = p.controller.IsFocused(p.window.Title())
					focusKnown = err == nil
				}
				if !popupShouldDismiss(focused, focusKnown) {
					focusLostAt = time.Time{}
					continue
				}
				if focusLostAt.IsZero() {
					focusLostAt = now
				}
				if !popupFocusLossReady(focusLostAt, now) {
					continue
				}

				shouldPaste := p.consumeAutoPaste()
				if shouldPaste && p.paster != nil {
					if err := p.paster.Paste(); err != nil {
						fyne.Do(func() { p.status.SetText("Não foi possível colar automaticamente") })
					}
				}
				fyne.Do(func() { p.Hide() })
				return
			}
		}
	}()
}

func popupDismissalReady(shownAt, now time.Time) bool {
	return now.Sub(shownAt) >= popupMinimumVisibleDuration
}

func popupFocusLossReady(lostAt, now time.Time) bool {
	return !lostAt.IsZero() && now.Sub(lostAt) >= popupFocusLossGrace
}

func (p *Popup) armAutoPaste() {
	p.pasteMu.Lock()
	p.pasteArmed = true
	p.pasteMu.Unlock()
}

func (p *Popup) disarmAutoPaste() {
	p.pasteMu.Lock()
	p.pasteArmed = false
	p.pasteMu.Unlock()
}

func (p *Popup) autoPasteArmed() bool {
	p.pasteMu.Lock()
	defer p.pasteMu.Unlock()
	return p.pasteArmed
}

func (p *Popup) consumeAutoPaste() bool {
	p.pasteMu.Lock()
	defer p.pasteMu.Unlock()
	armed := p.pasteArmed
	p.pasteArmed = false
	return armed
}

func (p *Popup) stopDismissalMonitor() {
	p.dismissMu.Lock()
	stop := p.dismissStop
	p.dismissStop = nil
	p.dismissMu.Unlock()
	if stop != nil {
		close(stop)
	}
}

func (p *Popup) hideWindowDecorations() {
	if p.controller == nil {
		return
	}

	title := p.window.Title()
	go func() {
		for attempt := 0; attempt < 4; attempt++ {
			if err := p.controller.HideDecorations(title); err == nil {
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
	}()
}

func popupShouldDismiss(focused, focusKnown bool) bool {
	return focusKnown && !focused
}

func (p *Popup) refreshItems() {
	p.items = p.manager.History().Items()
	for id := range p.items {
		p.list.SetItemHeight(id, historyRowHeight)
	}
	if len(p.items) == 0 {
		p.status.SetText("Histórico vazio")
		p.list.Hide()
		p.emptyState.Show()
	} else {
		p.status.SetText(historyStatus(len(p.items)))
		p.emptyState.Hide()
		p.list.Show()
	}
	// Visibility changes do not automatically re-run a parent Stack layout.
	// Resize keeps the list rows attached to the visible popup body after an
	// empty history becomes populated.
	if p.body != nil {
		p.body.Resize(p.body.Size())
	}
	p.list.Refresh()
}

func historyStatus(count int) string {
	itemLabel := "itens"
	if count == 1 {
		itemLabel = "item"
	}

	return fmt.Sprintf("%d %s — Enter ou clique para copiar", count, itemLabel)
}

func (p *Popup) selectItem(id widget.ListItemID) {
	if id < 0 || id >= len(p.items) {
		return
	}

	if err := p.manager.Select(p.items[id].ID); err != nil {
		p.status.SetText("Não foi possível copiar o item")
		return
	}
	if p.items[id].Kind == core.ClipKindImage {
		p.disarmAutoPaste()
		p.status.SetText("Imagem copiada — cole onde preferir")
		return
	}
	p.armAutoPaste()
	p.status.SetText("Item copiado — clique em um campo para colar")
}
