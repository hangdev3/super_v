package fyneui

import "testing"

func TestPreviewTextNormalizesWhitespaceAndLimitsLongClipboardContent(t *testing.T) {
	// Arrange
	text := "linha um\nlinha dois\nlinha tres"

	// Act
	preview := previewText(text, 18)

	// Assert
	if preview != "linha um linha..." {
		t.Fatalf("expected compact preview, got %q", preview)
	}
}
