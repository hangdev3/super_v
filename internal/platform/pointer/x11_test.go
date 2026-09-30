package pointer

import (
	"testing"

	"github.com/BurntSushi/xgb/xproto"
)

func TestPositionFromReplyUsesRootCoordinates(t *testing.T) {
	// Arrange
	reply := xproto.QueryPointerReply{RootX: 640, RootY: 360}

	// Act
	position := positionFromReply(reply)

	// Assert
	if position.X != 640 || position.Y != 360 {
		t.Fatalf("expected root pointer position (640, 360), got %#v", position)
	}
}

func TestValidateX11EnvironmentRejectsWayland(t *testing.T) {
	// Arrange

	// Act
	err := validateX11Environment("wayland", ":0")

	// Assert
	if err == nil {
		t.Fatalf("expected Wayland to be rejected by the X11-only MVP")
	}
}

func TestValidateX11EnvironmentAcceptsX11WithDisplay(t *testing.T) {
	// Arrange

	// Act
	err := validateX11Environment("x11", ":0")

	// Assert
	if err != nil {
		t.Fatalf("expected X11 environment to be accepted, got %v", err)
	}
}
