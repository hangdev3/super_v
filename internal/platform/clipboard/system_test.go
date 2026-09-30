package clipboard

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestImageClipboardArgsUsePNGSelection(t *testing.T) {
	got := imageClipboardArgs("/tmp/capture.png")
	want := []string{"-selection", "clipboard", "-target", "image/png", "-i", "/tmp/capture.png"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("image clipboard args = %#v, want %#v", got, want)
	}
}

func TestReadTextWithXclipRejectsImageSelectionsBeforeReadingText(t *testing.T) {
	// Arrange
	calls := 0
	run := func(_ string, args ...string) ([]byte, error) {
		calls++
		if !reflect.DeepEqual(args, xclipTargetsArgs()) {
			t.Fatalf("unexpected xclip call arguments: %#v", args)
		}
		return []byte("TARGETS\nimage/png\nUTF8_STRING\n"), nil
	}

	// Act
	_, err := readTextWithXclip("xclip", run)

	// Assert
	if !errors.Is(err, ErrClipboardNotText) {
		t.Fatalf("expected image clipboard to be ignored as non-text, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected no text read after detecting image content, got %d calls", calls)
	}
}

func TestReadTextWithXclipReadsTheAdvertisedUTF8Target(t *testing.T) {
	// Arrange
	var calls [][]string
	run := func(_ string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		if reflect.DeepEqual(args, xclipTargetsArgs()) {
			return []byte("TARGETS\nUTF8_STRING\nTEXT\n"), nil
		}
		if reflect.DeepEqual(args, xclipTextArgs("UTF8_STRING")) {
			return []byte("texto legítimo"), nil
		}
		t.Fatalf("unexpected xclip call arguments: %#v", args)
		return nil, nil
	}

	// Act
	got, err := readTextWithXclip("xclip", run)

	// Assert
	if err != nil {
		t.Fatalf("expected textual clipboard to be read, got %v", err)
	}
	if got != "texto legítimo" {
		t.Fatalf("clipboard text = %q, want %q", got, "texto legítimo")
	}
	wantCalls := [][]string{xclipTargetsArgs(), xclipTextArgs("UTF8_STRING")}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("xclip calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestReadImageWithXclipPersistsImageAndReturnsAStablePath(t *testing.T) {
	// Arrange
	directory := t.TempDir()
	data := []byte("fake png bytes")
	run := func(_ string, args ...string) ([]byte, error) {
		if reflect.DeepEqual(args, xclipTargetsArgs()) {
			return []byte("TARGETS\nimage/png\n"), nil
		}
		if reflect.DeepEqual(args, xclipTextArgs("image/png")) {
			return data, nil
		}
		t.Fatalf("unexpected xclip call arguments: %#v", args)
		return nil, nil
	}

	// Act
	path, found, err := readImageWithXclip("xclip", directory, run)
	secondPath, secondFound, secondErr := readImageWithXclip("xclip", directory, run)

	// Assert
	if err != nil || secondErr != nil {
		t.Fatalf("expected image clipboard reads to succeed, err=%v secondErr=%v", err, secondErr)
	}
	if !found || !secondFound || path != secondPath {
		t.Fatalf("expected a stable image path, path=%q secondPath=%q found=%v secondFound=%v", path, secondPath, found, secondFound)
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("image path extension = %q, want .png", filepath.Ext(path))
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read persisted clipboard image: %v", err)
	}
	if !reflect.DeepEqual(stored, data) {
		t.Fatalf("persisted image bytes = %q, want %q", stored, data)
	}
}

func TestReadImageWithXclipLeavesTextSelectionsForTheTextReader(t *testing.T) {
	// Arrange
	run := func(_ string, args ...string) ([]byte, error) {
		if !reflect.DeepEqual(args, xclipTargetsArgs()) {
			t.Fatalf("unexpected xclip call arguments: %#v", args)
		}
		return []byte("TARGETS\nUTF8_STRING\n"), nil
	}

	// Act
	path, found, err := readImageWithXclip("xclip", t.TempDir(), run)

	// Assert
	if err != nil {
		t.Fatalf("expected text target not to be an image error, got %v", err)
	}
	if found || path != "" {
		t.Fatalf("expected text selection to remain outside the image reader, path=%q found=%v", path, found)
	}
}
