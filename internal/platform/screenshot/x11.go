package screenshot

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var ErrCaptureCancelled = errors.New("screenshot capture cancelled")

type Capturer interface {
	Capture() (string, error)
}

type X11Screenshot struct {
	directory string
	run       func(outputPath string) error
	now       func() time.Time
}

func NewX11Screenshot() (*X11Screenshot, error) {
	command, err := exec.LookPath("gnome-screenshot")
	if err != nil {
		return nil, errors.New("captura de tela indisponivel: instale gnome-screenshot")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("obter diretorio pessoal para capturas: %w", err)
	}

	return newX11Screenshot(
		filepath.Join(home, "Imagens", "Captures"),
		func(outputPath string) error {
			if err := exec.Command(command, "-a", "-f", outputPath).Run(); err != nil {
				var exitError *exec.ExitError
				if errors.As(err, &exitError) {
					return ErrCaptureCancelled
				}
				return fmt.Errorf("executar gnome-screenshot: %w", err)
			}
			return nil
		},
		time.Now,
	), nil
}

func newX11Screenshot(directory string, run func(outputPath string) error, now func() time.Time) *X11Screenshot {
	return &X11Screenshot{
		directory: directory,
		run:       run,
		now:       now,
	}
}

func (s *X11Screenshot) Capture() (string, error) {
	if err := os.MkdirAll(s.directory, 0o755); err != nil {
		return "", fmt.Errorf("criar diretorio de capturas: %w", err)
	}

	outputPath := filepath.Join(s.directory, fmt.Sprintf("capture-%s.png", s.now().Format("20060102-150405.000000000")))
	if err := s.run(outputPath); err != nil {
		return "", err
	}
	if _, err := os.Stat(outputPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrCaptureCancelled
		}
		return "", fmt.Errorf("verificar captura salva: %w", err)
	}

	return outputPath, nil
}
