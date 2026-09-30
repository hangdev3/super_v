package input

import (
	"testing"

	"github.com/BurntSushi/xgb/xproto"
)

func TestKeycodeForKeysymFindsTheFirstMatchingKeycode(t *testing.T) {
	// Arrange
	firstKeycode := xproto.Keycode(8)
	keysymsPerKeycode := byte(2)
	keysyms := []xproto.Keysym{
		0x0076, 0x0056,
		0x0063, 0x0043,
		0xffe3, 0xffe4,
	}

	// Act
	vKeycode, foundV := keycodeForKeysym(firstKeycode, keysyms, keysymsPerKeycode, 0x0076, 0x0056)
	controlKeycode, foundControl := keycodeForKeysym(firstKeycode, keysyms, keysymsPerKeycode, 0xffe3, 0xffe4)

	// Assert
	if !foundV || vKeycode != 8 {
		t.Fatalf("V keycode = %d, found=%v, want 8/true", vKeycode, foundV)
	}
	if !foundControl || controlKeycode != 10 {
		t.Fatalf("Control keycode = %d, found=%v, want 10/true", controlKeycode, foundControl)
	}
	if _, found := keycodeForKeysym(firstKeycode, keysyms, keysymsPerKeycode, 0xffff); found {
		t.Fatalf("expected an unknown keysym not to be found")
	}
}
