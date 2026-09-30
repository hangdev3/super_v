package main

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"super_v/internal/core"
	hotkeyplatform "super_v/internal/platform/hotkey"
	screenshotplatform "super_v/internal/platform/screenshot"
)

func TestShouldShowPopupOnStartOnlyForExplicitDiagnosticFlag(t *testing.T) {
	// Arrange
	argsWithoutFlag := []string{"super-v"}
	argsWithFlag := []string{"super-v", "--show-on-start"}

	// Act
	withoutFlag := shouldShowPopupOnStart(argsWithoutFlag)
	withFlag := shouldShowPopupOnStart(argsWithFlag)

	// Assert
	if withoutFlag {
		t.Fatalf("expected normal startup not to show the popup")
	}
	if !withFlag {
		t.Fatalf("expected --show-on-start to request the diagnostic popup")
	}
}

func TestShortcutDebounceIgnoresAutoRepeat(t *testing.T) {
	// Arrange
	start := time.Unix(100, 0)

	// Act
	first := shouldHandleShortcut(time.Time{}, start)
	repeat := shouldHandleShortcut(start, start.Add(100*time.Millisecond))
	nextPress := shouldHandleShortcut(start, start.Add(300*time.Millisecond))

	// Assert
	if !first {
		t.Fatalf("expected the first shortcut event to be handled")
	}
	if repeat {
		t.Fatalf("expected an auto-repeat event to be ignored")
	}
	if !nextPress {
		t.Fatalf("expected a later shortcut press to be handled")
	}
}

func TestFirstShortcutEventDispatchesPopupOnce(t *testing.T) {
	// Arrange
	lastShortcut := time.Time{}
	opened := 0
	now := time.Unix(200, 0)

	// Act
	handled := dispatchShortcut(&lastShortcut, now, func() { opened++ })

	// Assert
	if !handled {
		t.Fatalf("expected the first shortcut event to be handled")
	}
	if opened != 1 {
		t.Fatalf("expected the first shortcut event to open once, got %d", opened)
	}
}

func TestWaitForShutdownQuitsOnInterruptOrTermination(t *testing.T) {
	tests := []struct {
		name   string
		signal os.Signal
	}{
		{name: "interrupt", signal: os.Interrupt},
		{name: "termination", signal: syscall.SIGTERM},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Arrange
			signals := make(chan os.Signal, 1)
			quitCalled := make(chan struct{})

			// Act
			go waitForShutdown(signals, func() { close(quitCalled) })
			signals <- testCase.signal

			// Assert
			select {
			case <-quitCalled:
			case <-time.After(time.Second):
				t.Fatalf("expected %s to request application shutdown", testCase.name)
			}
		})
	}
}

type fakeScreenshotCapturer struct {
	path string
	err  error
}

func (f *fakeScreenshotCapturer) Capture() (string, error) {
	return f.path, f.err
}

type fakeScreenshotNotifier struct {
	title   string
	message string
	calls   int
	err     error
}

func (f *fakeScreenshotNotifier) Notify(title, message string) error {
	f.calls++
	f.title = title
	f.message = message
	return f.err
}

type fakeScreenshotClipboard struct {
	images []string
}

func (f *fakeScreenshotClipboard) ReadText() (string, error) { return "", nil }

func (f *fakeScreenshotClipboard) WriteText(string) error { return nil }

func (f *fakeScreenshotClipboard) WriteImage(path string) error {
	f.images = append(f.images, path)
	return nil
}

func TestCaptureScreenshotsStoresImageAndNotifiesAfterSelection(t *testing.T) {
	// Arrange
	events := make(chan hotkeyplatform.Event, 1)
	events <- hotkeyplatform.Event{}
	close(events)
	clipboard := &fakeScreenshotClipboard{}
	manager := core.NewManager(core.NewHistory(5), clipboard)
	capturer := &fakeScreenshotCapturer{path: "/home/look/Imagens/Captures/capture.png"}
	notifier := &fakeScreenshotNotifier{}

	// Act
	captureScreenshots(events, capturer, notifier, manager)

	// Assert
	items := manager.History().Items()
	if len(items) != 1 || items[0].Kind != core.ClipKindImage || items[0].ImagePath != capturer.path {
		t.Fatalf("expected captured image in history, got %#v", items)
	}
	if len(clipboard.images) != 1 || clipboard.images[0] != capturer.path {
		t.Fatalf("expected newest captured image in clipboard, got %#v", clipboard.images)
	}
	if notifier.calls != 1 {
		t.Fatalf("expected one screenshot notification, got %d", notifier.calls)
	}
	if notifier.title != "Captura de tela salva" || notifier.message != capturer.path {
		t.Fatalf("unexpected notification: %q / %q", notifier.title, notifier.message)
	}
}

func TestCaptureScreenshotsIgnoresCancelledSelection(t *testing.T) {
	// Arrange
	events := make(chan hotkeyplatform.Event, 1)
	events <- hotkeyplatform.Event{}
	close(events)
	manager := core.NewManager(core.NewHistory(5), &fakeScreenshotClipboard{})
	capturer := &fakeScreenshotCapturer{err: screenshotplatform.ErrCaptureCancelled}
	notifier := &fakeScreenshotNotifier{}

	// Act
	captureScreenshots(events, capturer, notifier, manager)

	// Assert
	if len(manager.History().Items()) != 0 {
		t.Fatalf("expected cancelled capture not to enter history")
	}
	if notifier.calls != 0 {
		t.Fatalf("expected cancelled capture not to notify, got %d calls", notifier.calls)
	}
}

func TestCaptureScreenshotsContinuesAfterCaptureError(t *testing.T) {
	// Arrange
	events := make(chan hotkeyplatform.Event, 1)
	events <- hotkeyplatform.Event{}
	close(events)
	manager := core.NewManager(core.NewHistory(5), &fakeScreenshotClipboard{})
	wantErr := errors.New("selector failed")
	capturer := &fakeScreenshotCapturer{err: wantErr}
	notifier := &fakeScreenshotNotifier{}

	// Act
	captureScreenshots(events, capturer, notifier, manager)

	// Assert
	if len(manager.History().Items()) != 0 {
		t.Fatalf("expected failed capture not to enter history")
	}
	if notifier.calls != 0 {
		t.Fatalf("expected failed capture not to notify, got %d calls", notifier.calls)
	}
}
