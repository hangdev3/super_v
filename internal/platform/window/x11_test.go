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

func TestActiveWindowMessageUsesTheKeyPressTimestamp(t *testing.T) {
	// Arrange
	target := xproto.Window(0x33)
	active := xproto.Window(0x55)
	atom := xproto.Atom(0x77)
	timestamp := uint32(123456)

	// Act
	message := activeWindowMessage(target, atom, timestamp, active)

	// Assert
	if message.Format != 32 || message.Window != target || message.Type != atom {
		t.Fatalf("unexpected active window message header: %#v", message)
	}
	wantData := []uint32{1, timestamp, uint32(active), 0, 0}
	if !reflect.DeepEqual(message.Data.Data32, wantData) {
		t.Fatalf("active window message data = %#v, want %#v", message.Data.Data32, wantData)
	}
}
