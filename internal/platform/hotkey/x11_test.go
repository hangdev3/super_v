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

func TestShortcutModifierMasksPreserveRequiredModifiersWithLockStates(t *testing.T) {
	// Arrange
	required := x11Mod4Mask | x11ShiftMask

	// Act
	masks := shortcutModifierMasks(required)

	// Assert
	if len(masks) != 8 {
		t.Fatalf("expected 8 modifier combinations, got %d", len(masks))
	}
	for _, mask := range masks {
		if mask&required != required {
			t.Fatalf("mask %#x lost required modifiers %#x", mask, required)
		}
	}
}

func TestNewSuperShiftSUsesShiftAndS(t *testing.T) {
	shortcut := NewSuperShiftS()

	if shortcut.keysym != xproto.Keysym(0x0073) {
		t.Fatalf("Super+Shift+S keysym = %#x, want %#x", shortcut.keysym, 0x0073)
	}
	if shortcut.requiredModifiers != x11Mod4Mask|x11ShiftMask {
		t.Fatalf("Super+Shift+S modifiers = %#x, want %#x", shortcut.requiredModifiers, x11Mod4Mask|x11ShiftMask)
	}
}

func TestNewSuperShiftVUsesShiftAndV(t *testing.T) {
	shortcut := NewSuperShiftV()

	if shortcut.keysym != xproto.Keysym(0x0076) {
		t.Fatalf("Super+Shift+V keysym = %#x, want %#x", shortcut.keysym, 0x0076)
	}
	if shortcut.requiredModifiers != x11Mod4Mask|x11ShiftMask {
		t.Fatalf("Super+Shift+V modifiers = %#x, want %#x", shortcut.requiredModifiers, x11Mod4Mask|x11ShiftMask)
	}
	if shortcut.displayName != "Super+Shift+V" {
		t.Fatalf("Super+Shift+V display name = %q", shortcut.displayName)
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

func TestShortcutEventCarriesKeyPressTimestamp(t *testing.T) {
	// Arrange
	wantTimestamp := xproto.Timestamp(123456)

	// Act
	event, ok := shortcutEvent(xproto.KeyPressEvent{Detail: 55, Time: wantTimestamp}, 55)

	// Assert
	if !ok {
		t.Fatalf("expected the matching key press to produce a shortcut event")
	}
	if event.Timestamp != uint32(wantTimestamp) {
		t.Fatalf("shortcut timestamp = %d, want %d", event.Timestamp, wantTimestamp)
	}
}

func TestShortcutEventIgnoresOtherKeyPresses(t *testing.T) {
	if isShortcutKeyPress(xproto.KeyPressEvent{Detail: 56}, 55) {
		t.Fatalf("expected a key press for another keycode to be ignored")
	}
}
