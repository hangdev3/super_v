# Changelog

All notable changes to Super V are documented here.

## [0.1.0] — 2026-09-30

### Added

- Global `Super+Shift+V` shortcut for opening the clipboard history popup.
- Global `Super+Shift+S` shortcut for selecting a screen area with
  `gnome-screenshot`.
- Automatic storage of screenshots in `~/Imagens/Captures`.
- Desktop notification after a screenshot is saved.
- Immediate publication of the newest screenshot to the X11 image clipboard,
  so the next paste uses the capture just taken.
- Mixed text/image history with thumbnails and a maximum of 10 entries.
- Explicit image recovery to the clipboard without text auto-paste.
- X11 MIME detection that prevents PNG metadata or hashes from becoming text
  history entries.
- Contextual popup improvements: 64 px rows, 48 px thumbnails, rounded
  styling and approximately 90% background opacity.
- System tray icon with only the `Sair` action and no auxiliary window.
- Focus/timestamp handling for the first global shortcut activation.
- Debian packaging with runtime dependencies, application menu entry and
  Cinnamon autostart.
- Apache License 2.0 for open-source distribution.

### Fixed

- Prevented screenshot paste from using the previous clipboard image.
- Prevented duplicate image rows caused by clipboard echoes.
- Prevented blank history rows caused by custom Fyne list wrappers.
- Prevented test fixtures based on infinite `image.Uniform` values from
  causing extreme memory consumption during PNG encoding.

### Validation

- `go test -count=1 ./...`
- `CGO_ENABLED=0 go test -tags ci -count=1 ./...`
- `go test -race -count=1 ./...`
- `go vet ./...`
- `make package-test`

### Known limitations

- The first release targets Linux Mint Cinnamon on X11.
- Wayland global shortcuts and image auto-paste are not part of this release.
- The screenshot and clipboard integrations require the packages declared by
  the Debian package.
