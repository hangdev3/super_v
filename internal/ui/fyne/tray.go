package fyneui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// SetupTray registers the status icon and its exit-only menu. It deliberately
// does not register a tray window, so clicking the icon cannot open an empty or
// unrelated Fyne screen. The function returns false when the current Fyne
// driver has no system tray support.
func SetupTray(application fyne.App) bool {
	host, ok := application.(desktop.App)
	if !ok {
		return false
	}

	quit := func() { fyne.Do(application.Quit) }
	configureTray(host, quit)
	return true
}

func configureTray(host desktop.App, onQuit func()) {
	host.SetSystemTrayMenu(newTrayMenu(onQuit))
	host.SetSystemTrayIcon(compactTrayIcon())
}

func compactTrayIcon() fyne.Resource {
	const size = 16
	icon := image.NewRGBA(image.Rect(0, 0, size, size))
	white := image.NewUniform(color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	green := image.NewUniform(color.NRGBA{R: 92, G: 214, B: 125, A: 255})

	// Draw a one-pixel clipboard outline that remains legible in a 16x16 tray slot.
	draw.Draw(icon, image.Rect(4, 4, 12, 14), white, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(5, 5, 11, 13), image.Transparent, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(6, 2, 10, 5), white, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(7, 3, 9, 4), image.Transparent, image.Point{}, draw.Src)

	// A small status dot communicates that the resident process is active.
	draw.Draw(icon, image.Rect(11, 11, 15, 15), green, image.Point{}, draw.Src)
	draw.Draw(icon, image.Rect(12, 12, 14, 14), image.Transparent, image.Point{}, draw.Src)

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, icon); err != nil {
		return fyne.NewStaticResource("super-v-tray.png", nil)
	}
	return fyne.NewStaticResource("super-v-tray.png", encoded.Bytes())
}

func newTrayMenu(onQuit func()) *fyne.Menu {
	exit := fyne.NewMenuItem("Sair", onQuit)
	exit.Icon = theme.LogoutIcon()
	return fyne.NewMenu("Super+Shift+V", exit)
}
