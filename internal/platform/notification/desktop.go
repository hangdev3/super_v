package notification

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Notifier interface {
	Notify(title, message string) error
}

type DesktopNotifier struct {
	command string
	run     func(name string, args ...string) error
}

func NewNotifySend() (*DesktopNotifier, error) {
	command, err := exec.LookPath("notify-send")
	if err != nil {
		return nil, errors.New("notificacao desktop indisponivel: instale libnotify-bin")
	}

	return newNotifySend(command, func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	}), nil
}

func newNotifySend(command string, run func(name string, args ...string) error) *DesktopNotifier {
	return &DesktopNotifier{command: command, run: run}
}

func (n *DesktopNotifier) Notify(title, message string) error {
	if strings.TrimSpace(title) == "" || strings.TrimSpace(message) == "" {
		return errors.New("notificacao desktop exige titulo e mensagem")
	}

	if err := n.run(n.command, "--app-name=Super+V", title, message); err != nil {
		return fmt.Errorf("enviar notificacao desktop: %w", err)
	}
	return nil
}
