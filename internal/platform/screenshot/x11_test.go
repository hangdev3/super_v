package screenshot

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestX11ScreenshotStoresCapturedPNGInConfiguredDirectory(t *testing.T) {
	// Arrange
	directory := t.TempDir()
	service := newX11Screenshot(
		directory,
		func(path string) error {
			return os.WriteFile(path, []byte("png"), 0600)
		},
		func() time.Time { return time.Date(2026, time.September, 30, 3, 10, 11, 123456789, time.UTC) },
	)

	// Act
	path, err := service.Capture()

	// Assert
	if err != nil {
		t.Fatalf("expected screenshot to be stored, got %v", err)
	}
	if filepath.Dir(path) != directory {
		t.Fatalf("screenshot directory = %q, want %q", filepath.Dir(path), directory)
	}
	if !strings.HasPrefix(filepath.Base(path), "capture-20260930-031011") || filepath.Ext(path) != ".png" {
		t.Fatalf("unexpected screenshot filename %q", filepath.Base(path))
	}
}

func TestX11ScreenshotReturnsCancellationWhenNoFileIsCreated(t *testing.T) {
	// Arrange
	service := newX11Screenshot(t.TempDir(), func(string) error { return nil }, time.Now)

	// Act
	_, err := service.Capture()

	// Assert
	if !errors.Is(err, ErrCaptureCancelled) {
		t.Fatalf("expected capture cancellation, got %v", err)
	}
}

func TestX11ScreenshotPropagatesCaptureErrors(t *testing.T) {
	// Arrange
	wantErr := errors.New("selector failed")
	service := newX11Screenshot(t.TempDir(), func(string) error { return wantErr }, time.Now)

	// Act
	_, err := service.Capture()

	// Assert
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected capture error %v, got %v", wantErr, err)
	}
}
