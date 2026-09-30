package notification

import (
	"errors"
	"reflect"
	"testing"
)

var errNotificationCommand = errors.New("notification command failed")

func TestNotifySendBuildsDesktopNotificationArguments(t *testing.T) {
	// Arrange
	var command string
	var args []string
	service := newNotifySend("notify-send", func(name string, received ...string) error {
		command = name
		args = received
		return nil
	})

	// Act
	err := service.Notify("Captura de tela salva", "/home/look/Imagens/Captures/capture.png")

	// Assert
	if err != nil {
		t.Fatalf("expected notification to succeed, got %v", err)
	}
	if command != "notify-send" {
		t.Fatalf("notification command = %q, want notify-send", command)
	}
	want := []string{"--app-name=Super+V", "Captura de tela salva", "/home/look/Imagens/Captures/capture.png"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("notification args = %#v, want %#v", args, want)
	}
}

func TestNotifySendPropagatesCommandErrors(t *testing.T) {
	// Arrange
	wantErr := errNotificationCommand
	service := newNotifySend("notify-send", func(string, ...string) error {
		return wantErr
	})

	// Act
	err := service.Notify("Captura de tela salva", "capture.png")

	// Assert
	if err == nil || err.Error() != "enviar notificacao desktop: notification command failed" {
		t.Fatalf("expected wrapped notification error, got %v", err)
	}
}

func TestNotifySendRejectsBlankTitleOrMessage(t *testing.T) {
	// Arrange
	service := newNotifySend("notify-send", func(string, ...string) error {
		t.Fatal("expected command not to run for invalid notification")
		return nil
	})

	// Act
	titleErr := service.Notify(" ", "capture.png")
	messageErr := service.Notify("Captura de tela salva", " ")

	// Assert
	if titleErr == nil || messageErr == nil {
		t.Fatalf("expected blank notification fields to be rejected")
	}
}
