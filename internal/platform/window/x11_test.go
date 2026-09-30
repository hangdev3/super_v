package window

import (
	"reflect"
	"testing"

	"github.com/BurntSushi/xgb/xproto"
)

func TestMotifNoDecorationHintsDisableWindowChrome(t *testing.T) {
	// Arrange
	want := []byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// Act
	got := motifNoDecorationHints()

	// Assert
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Motif decoration hints = %#v, want %#v", got, want)
	}
}

func TestWindowIDsFromClientListProperty(t *testing.T) {
	// Arrange
	property := []byte{
		0x04, 0x03, 0x02, 0x01,
		0xd0, 0xc0, 0xb0, 0xa0,
	}

	// Act
	got := windowIDsFromProperty(property)

	// Assert
	want := []xproto.Window{0x01020304, 0xa0b0c0d0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("client window IDs = %#v, want %#v", got, want)
	}
}
