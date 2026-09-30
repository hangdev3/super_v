package main

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestShouldShowPopupOnStartOnlyForExplicitDiagnosticFlag(t *testing.T) {
	// Arrange
	argsWithoutFlag := []string{"super-v"}
	argsWithFlag := []string{"super-v", "--show-on-start"}

	// Act
	withoutFlag := shouldShowPopupOnStart(argsWithoutFlag)
	withFlag := shouldShowPopupOnStart(argsWithFlag)

	// Assert
	if withoutFlag {
		t.Fatalf("expected normal startup not to show the popup")
	}
	if !withFlag {
		t.Fatalf("expected --show-on-start to request the diagnostic popup")
	}
}

func TestShortcutDebounceIgnoresAutoRepeat(t *testing.T) {
	// Arrange
	start := time.Unix(100, 0)

	// Act
	first := shouldHandleShortcut(time.Time{}, start)
	repeat := shouldHandleShortcut(start, start.Add(100*time.Millisecond))
	nextPress := shouldHandleShortcut(start, start.Add(300*time.Millisecond))

	// Assert
	if !first {
		t.Fatalf("expected the first shortcut event to be handled")
	}
	if repeat {
		t.Fatalf("expected an auto-repeat event to be ignored")
	}
	if !nextPress {
		t.Fatalf("expected a later shortcut press to be handled")
	}
}

func TestFirstShortcutEventDispatchesPopupOnce(t *testing.T) {
	// Arrange
	lastShortcut := time.Time{}
	opened := 0
	now := time.Unix(200, 0)

	// Act
	handled := dispatchShortcut(&lastShortcut, now, func() { opened++ })

	// Assert
	if !handled {
		t.Fatalf("expected the first shortcut event to be handled")
	}
	if opened != 1 {
		t.Fatalf("expected the first shortcut event to open once, got %d", opened)
	}
}

func TestWaitForShutdownQuitsOnInterruptOrTermination(t *testing.T) {
	tests := []struct {
		name   string
		signal os.Signal
	}{
		{name: "interrupt", signal: os.Interrupt},
		{name: "termination", signal: syscall.SIGTERM},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Arrange
			signals := make(chan os.Signal, 1)
			quitCalled := make(chan struct{})

			// Act
			go waitForShutdown(signals, func() { close(quitCalled) })
			signals <- testCase.signal

			// Assert
			select {
			case <-quitCalled:
			case <-time.After(time.Second):
				t.Fatalf("expected %s to request application shutdown", testCase.name)
			}
		})
	}
}
