package fyneui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
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
}

type Popup struct {
	manager     *core.Manager
	pointer     PointerSource
	window      fyne.Window
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

const (
	historyPreviewMaxRunes      = 48
	popupAnchorOffset           = 12
	popupMinimumVisibleDuration = 800 * time.Millisecond
	popupDismissPollPeriod      = 80 * time.Millisecond
)

func newHistoryItem() fyne.CanvasObject {
	label := widget.NewLabel("")
	label.Truncation = fyne.TextTruncateClip
	return label
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
			label := object.(*widget.Label)
			if id >= 0 && id < len(p.items) {
				label.SetText(previewText(p.items[id].Text, historyPreviewMaxRunes))
			}
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
	p.window = application.NewWindow("Super+V — histórico")
	body := container.NewStack(p.list, container.NewCenter(p.emptyState))
	p.window.SetContent(container.NewBorder(header, p.status, nil, nil, body))
	p.window.SetPadded(true)
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

func (p *Popup) ShowAtPointer() error {
	position, err := p.pointer.Position()
	if err != nil {
		return err
	}

	p.refreshItems()
	p.disarmAutoPaste()
	if desktopWindow, ok := p.window.(desktop.Window); ok {
		desktopWindow.RequestAlwaysOnTop()
		desktopWindow.RequestPosition(position.X+popupAnchorOffset, position.Y+popupAnchorOffset)
	}
	p.window.Show()
	p.window.RequestFocus()
	p.hideWindowDecorations()
	p.startDismissalMonitor()
	return nil
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
	go func() {
		ticker := time.NewTicker(popupDismissPollPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if !popupDismissalReady(shownAt, time.Now()) {
					continue
				}

				focused, focusKnown := true, false
				if p.controller != nil {
					var err error
					focused, err = p.controller.IsFocused(p.window.Title())
					focusKnown = err == nil
				}
				if !popupShouldDismiss(focused, focusKnown) {
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
	if len(p.items) == 0 {
		p.status.SetText("Histórico vazio")
		p.list.Hide()
		p.emptyState.Show()
	} else {
		p.status.SetText(historyStatus(len(p.items)))
		p.emptyState.Hide()
		p.list.Show()
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
	p.armAutoPaste()
	p.status.SetText("Item copiado — clique em um campo para colar")
}
