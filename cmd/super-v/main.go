package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"super_v/internal/core"
	clipboardplatform "super_v/internal/platform/clipboard"
	hotkeyplatform "super_v/internal/platform/hotkey"
	inputplatform "super_v/internal/platform/input"
	pointerplatform "super_v/internal/platform/pointer"
	windowplatform "super_v/internal/platform/window"
	"super_v/internal/ui/fyne"
)

const clipboardPollInterval = 350 * time.Millisecond
const shortcutDebounce = 250 * time.Millisecond

func main() {
	clipboard, err := clipboardplatform.NewSystemClipboard()
	if err != nil {
		log.Fatal(err)
	}

	pointer, err := pointerplatform.NewX11Pointer()
	if err != nil {
		log.Fatal(err)
	}
	defer pointer.Close()

	windowController, err := windowplatform.NewX11WindowController()
	if err != nil {
		log.Printf("decoracao nativa indisponivel: %v", err)
	} else {
		defer windowController.Close()
	}

	shortcut := hotkeyplatform.NewSuperV()
	if err := shortcut.Register(); err != nil {
		log.Fatal(err)
	}
	defer shortcut.Close()

	application := app.NewWithID("io.superv.clipboard")
	manager := core.NewManager(core.NewHistory(50), clipboard)
	popup := fyneui.NewPopup(application, manager, pointer)
	popup.SetWindowController(windowController)

	paster, err := inputplatform.NewX11Paster()
	if err != nil {
		log.Printf("auto-colagem indisponivel: %v", err)
	} else {
		defer paster.Close()
		popup.SetPaster(paster)
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)
	go waitForShutdown(shutdownSignals, func() {
		log.Printf("sinal de encerramento recebido; finalizando")
		fyne.Do(application.Quit)
	})

	go monitorClipboard(manager)
	go openPopupOnShortcut(shortcut, popup)

	log.Printf("super_v iniciado; Super+V registrado; aguardando atalho")
	if shouldShowPopupOnStart(os.Args) {
		time.AfterFunc(250*time.Millisecond, func() {
			fyne.Do(func() {
				if err := popup.ShowAtPointer(); err != nil {
					log.Printf("nao foi possivel abrir o popup de diagnostico: %v", err)
				}
			})
		})
	}

	application.Run()
}

func waitForShutdown(signals <-chan os.Signal, quit func()) {
	switch <-signals {
	case os.Interrupt, syscall.SIGTERM:
		quit()
	}
}

func monitorClipboard(manager *core.Manager) {
	ticker := time.NewTicker(clipboardPollInterval)
	defer ticker.Stop()

	var lastError string
	for range ticker.C {
		_, err := manager.SyncClipboard()
		if err == nil {
			lastError = ""
			continue
		}
		if err.Error() != lastError {
			log.Printf("clipboard indisponivel: %v", err)
			lastError = err.Error()
		}
	}
}

func openPopupOnShortcut(shortcut *hotkeyplatform.SuperV, popup *fyneui.Popup) {
	var lastShortcut time.Time
	for range shortcut.Events() {
		now := time.Now()
		if !dispatchShortcut(&lastShortcut, now, func() {
			log.Printf("Super+V recebido")
			fyne.DoAndWait(func() {
				if err := popup.ShowAtPointer(); err != nil {
					log.Printf("nao foi possivel abrir o historico: %v", err)
				}
			})
		}) {
			continue
		}
	}
}

func dispatchShortcut(lastShortcut *time.Time, current time.Time, open func()) bool {
	if !shouldHandleShortcut(*lastShortcut, current) {
		return false
	}
	*lastShortcut = current
	open()
	return true
}

func shouldHandleShortcut(lastShortcut, current time.Time) bool {
	return lastShortcut.IsZero() || current.Sub(lastShortcut) >= shortcutDebounce
}

func shouldShowPopupOnStart(args []string) bool {
	for _, arg := range args[1:] {
		if arg == "--show-on-start" {
			return true
		}
	}

	return false
}
