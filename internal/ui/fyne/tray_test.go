package fyneui

import (
	"bytes"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

type fakeTrayHost struct {
	menu      *fyne.Menu
	icon      fyne.Resource
	window    fyne.Window
	windowSet bool
}

func (f *fakeTrayHost) SetSystemTrayMenu(menu *fyne.Menu) {
	f.menu = menu
}

func (f *fakeTrayHost) SetSystemTrayIcon(icon fyne.Resource) {
	f.icon = icon
}

func (f *fakeTrayHost) SetSystemTrayWindow(window fyne.Window) {
	f.window = window
	f.windowSet = true
}

var _ desktop.App = (*fakeTrayHost)(nil)

func TestTrayMenuProvidesExitAction(t *testing.T) {
	// Arrange
	quitCalls := 0

	// Act
	menu := newTrayMenu(func() { quitCalls++ })

	// Assert
	if len(menu.Items) != 1 {
		t.Fatalf("expected one tray menu item, got %d", len(menu.Items))
	}
	if menu.Items[0].Label != "Sair" {
		t.Fatalf("tray menu item = %q, want %q", menu.Items[0].Label, "Sair")
	}
	menu.Items[0].Action()
	if quitCalls != 1 {
		t.Fatalf("expected tray exit action to be called once, got %d", quitCalls)
	}
}

func TestConfigureTrayRegistersOnlyMenuAndClipboardIcon(t *testing.T) {
	// Arrange
	host := &fakeTrayHost{}

	// Act
	configureTray(host, func() {})

	// Assert
	if host.menu == nil {
		t.Fatalf("expected tray menu to be registered")
	}
	if host.icon == nil || host.icon.Name() != compactTrayIcon().Name() {
		t.Fatalf("expected compact clipboard icon to be registered")
	}
	if host.windowSet {
		t.Fatalf("expected tray setup not to register a window opened by icon clicks")
	}
}

func TestCompactTrayIconUsesSmallRasterResource(t *testing.T) {
	// Arrange / Act
	icon := compactTrayIcon()
	image, err := png.Decode(bytes.NewReader(icon.Content()))

	// Assert
	if err != nil {
		t.Fatalf("expected tray icon to be a valid PNG: %v", err)
	}
	if image.Bounds().Dx() != 16 || image.Bounds().Dy() != 16 {
		t.Fatalf("tray icon size = %v, want 16x16", image.Bounds())
	}
}
