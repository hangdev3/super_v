package hotkey

import (
	"reflect"
	"testing"

	"github.com/BurntSushi/xgb/xproto"
)

func TestSuperVModifierMasksIncludeX11LockStates(t *testing.T) {
	want := []Modifier{
		x11Mod4Mask,
		x11Mod4Mask | x11LockMask,
		x11Mod4Mask | x11Mod2Mask,
		x11Mod4Mask | x11Mod2Mask | x11LockMask,
		x11Mod4Mask | x11Mod3Mask,
		x11Mod4Mask | x11LockMask | x11Mod3Mask,
		x11Mod4Mask | x11Mod2Mask | x11Mod3Mask,
		x11Mod4Mask | x11LockMask | x11Mod2Mask | x11Mod3Mask,
	}

	if got := superVModifierMasks(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Super+V modifier masks = %#v, want %#v", got, want)
	}
}

func TestShortcutEventOnlyDispatchesKeyPress(t *testing.T) {
	// Arrange
	press := xproto.KeyPressEvent{Detail: 55}
	release := xproto.KeyReleaseEvent{}

	// Act
	pressEvent := isShortcutKeyPress(press, 55)
	releaseEvent := isShortcutKeyPress(release, 55)

	// Assert
	if !pressEvent {
		t.Fatalf("expected a key press event to dispatch the shortcut")
	}
	if releaseEvent {
		t.Fatalf("expected a key release event not to dispatch the shortcut")
	}
}

func TestShortcutEventIgnoresOtherKeyPresses(t *testing.T) {
	if isShortcutKeyPress(xproto.KeyPressEvent{Detail: 56}, 55) {
		t.Fatalf("expected a key press for another keycode to be ignored")
	}
}
