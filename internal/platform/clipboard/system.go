package clipboard

import (
	"errors"
	"fmt"

	"github.com/atotto/clipboard"
)

type SystemClipboard struct{}

func NewSystemClipboard() (*SystemClipboard, error) {
	if clipboard.Unsupported {
		return nil, errors.New("clipboard indisponivel: instale xclip ou xsel no X11")
	}

	return &SystemClipboard{}, nil
}

func (SystemClipboard) ReadText() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("ler clipboard: %w", err)
	}

	return text, nil
}

func (SystemClipboard) WriteText(text string) error {
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("escrever clipboard: %w", err)
	}

	return nil
}
